package main

import (
	"flag"
	"fmt"
	"gophkeeper/cmd/client/internal/app"
	"log"
	"os"
)

func main() {
	versionFlag := flag.Bool("version", false, "Показать версию")
	helpFlag := flag.Bool("help", false, "Показать справку")

	flag.Parse()

	if *versionFlag {
		fmt.Println(app.VersionInfo())
		os.Exit(0)
	}

	if *helpFlag {
		fmt.Println("GophKeeper - безопасное хранение данных")
		fmt.Println("\nИспользование:")
		fmt.Println("  gophkeeper          Запуск клиента")
		fmt.Println("  gophkeeper -version Показать версию")
		fmt.Println("  gophkeeper -help    Показать эту справку")
		os.Exit(0)
	}

	gophkeeperApp := app.NewApp()

	if err := gophkeeperApp.Start(); err != nil {
		log.Fatal("Ошибка запуска приложения:", err)
	}
}
