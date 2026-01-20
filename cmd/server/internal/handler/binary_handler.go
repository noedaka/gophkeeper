package handler

import (
	"context"
	"errors"
	"gophkeeper/internal/proto"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) UploadBinary(stream proto.BinaryStorage_UploadBinaryServer) error {
	ctx := stream.Context()

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	var metadata string
	var recordID int
	var s3Key string
	var firstChunk = true

	// Pipe для потоковой записи в MinIO
	reader, writer := io.Pipe()

	// Канал для ошибки upload
	uploadErrChan := make(chan error, 1)

	// Запускаем upload только после получения s3Key
	var uploadStarted bool

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			// Клиент завершил отправку
			if !uploadStarted {
				writer.CloseWithError(errors.New("no chunks received"))
			} else {
				writer.Close()
			}
			break
		}
		if err != nil {
			writer.CloseWithError(err)
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		// Метаданные из первого чанка
		if firstChunk {
			if chunk.GetMetadata() == "" {
				writer.CloseWithError(errors.New("metadata required in first chunk"))
				return status.Errorf(codes.InvalidArgument, "metadata required in first chunk")
			}
			metadata = chunk.GetMetadata()

			// Создаём запись в БД
			var createErr error
			recordID, s3Key, createErr = h.BinaryService.Create(ctx, userID, metadata)
			if createErr != nil {
				writer.CloseWithError(createErr)
				return status.Errorf(codes.Internal, "failed to create record: %v", createErr)
			}

			// Теперь запускаем upload в MinIO
			go func() {
				defer writer.Close()
				uploadErrChan <- h.BinaryService.Upload(ctx, s3Key, reader, -1) // -1 = unknown size for streaming
			}()
			uploadStarted = true

			firstChunk = false
		}

		// Пишем чанк в pipe
		if _, writeErr := writer.Write(chunk.GetData()); writeErr != nil {
			return status.Errorf(codes.Internal, "failed to write chunk to pipe: %v", writeErr)
		}

		// Если последний чанк — закрываем writer (MinIO завершит)
		if chunk.GetIsLast() {
			writer.Close()
		}
	}

	// Ждём завершения upload в MinIO
	if uploadStarted {
		if uploadErr := <-uploadErrChan; uploadErr != nil {
			// Best effort cleanup БД
			_ = h.BinaryService.Delete(ctx, recordID, userID)
			return status.Errorf(codes.Internal, "MinIO upload failed: %v", uploadErr)
		}
	} else {
		return status.Errorf(codes.InvalidArgument, "no data received")
	}

	i32ID := int32(recordID)
	response := proto.BinaryRecordID_builder{
		Id: &i32ID,
	}

	return stream.SendAndClose(response.Build())
}

// DownloadBinary — потоковая скачка бинарного файла
func (h *Handler) DownloadBinary(req *proto.BinaryRecordID, stream proto.BinaryStorage_DownloadBinaryServer) error {
	ctx := stream.Context()

	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	recordID := int(req.GetId())
	s3Key, err := h.BinaryService.GetKey(ctx, recordID, userID)

	if err != nil || s3Key == "" {
		return status.Errorf(codes.NotFound, "record not found: %v", err)
	}

	// Получаем объект из MinIO
	objReader, _, err := h.BinaryService.Get(ctx, s3Key)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get object from MinIO: %v", err)
	}
	defer objReader.Close()

	// Читаем и стримим чанки клиенту (по 1-4 MB)
	const chunkSize = 4 * 1024 * 1024 // 4 MB
	buf := make([]byte, chunkSize)
	sequence := int32(0)

	for {
		n, readErr := objReader.Read(buf)
		if n > 0 {
			chunkData := make([]byte, n)
			copy(chunkData, buf[:n])

			isLast := readErr == io.EOF
			chunk := proto.BinaryChunk_builder{
				Data:     chunkData,
				Sequence: &sequence,
				IsLast:   &isLast,
			}

			if sendErr := stream.Send(chunk.Build()); sendErr != nil {
				return status.Errorf(codes.Internal, "failed to send chunk: %v", sendErr)
			}

			sequence++
		}

		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return status.Errorf(codes.Internal, "failed to read object: %v", readErr)
		}
	}

	return nil
}

// ListBinaries — список бинарных записей пользователя
func (h *Handler) ListBinaries(ctx context.Context, _ *proto.BinaryEmpty) (*proto.BinaryRecordList, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	records, err := h.BinaryService.List(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot list binary records: %v", err)
	}

	response := proto.BinaryRecordList_builder{
		Records: make([]*proto.BinaryRecordInfo, 0, len(records)),
	}

	for _, rec := range records {
		i32ID := int32(rec.ID)
		info := proto.BinaryRecordInfo_builder{
			Id:       &i32ID,
			Metadata: &rec.Metadata,
		}
		response.Records = append(response.Records, info.Build())
	}

	return response.Build(), nil
}

// DeleteBinary — удаление бинарной записи
func (h *Handler) DeleteBinary(ctx context.Context, req *proto.BinaryRecordID) (*proto.BinaryEmpty, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unauthenticated")
	}

	err := h.BinaryService.Delete(ctx, int(req.GetId()), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot delete binary record: %v", err)
	}

	response := proto.BinaryEmpty_builder{}

	return response.Build(), nil
}
