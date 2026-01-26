package main

import (
	"flag"
	"fmt"
	"gophkeeper/cmd/client/internal/app"
	"log"
	"os"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	versionFlag := flag.Bool("version", false, "Показать версию")
	helpFlag := flag.Bool("help", false, "Показать справку")

	flag.Parse()
	if *versionFlag {
		fmt.Printf("GophKeeper Client\n")
		fmt.Printf("Version: %s\n", buildVersion)
		fmt.Printf("Build date: %s\n", buildDate)
		fmt.Printf("Commit: %s\n", buildCommit)
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
