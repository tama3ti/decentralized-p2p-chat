package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/piheta/sept/backend/controllers"
	"github.com/piheta/sept/backend/db"
	"github.com/piheta/sept/backend/repos"
	"github.com/piheta/sept/backend/services"
)

//go:embed all:frontend/dist
var assets embed.FS

func initDataDir() {
	execPath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}
	execDir := filepath.Dir(execPath)

	// Create the folder inside the Resources directory of the app bundle
	resourcesPath := filepath.Join(execDir, "..", "Resources", "Data")
	err = os.MkdirAll(resourcesPath, 0o755)
	if err != nil {
		fmt.Println("Error creating folder:", err)
	}
	db.SEPT_DATA = resourcesPath
	fmt.Printf(db.SEPT_DATA)
}

func main() {
	// find current dir and create a Data dir for db and keypair
	initDataDir()

	//
	// startup logic
	// try to init db with jwt
	// if this fails, init repos with null db
	// on login, repos will correctly init the db from inside of "app" controller
	//

	user_repo := repos.NewUserRepo(db.DB)
	chat_repo := repos.NewChatRepo(db.DB)
	userchat_repo := repos.NewUserchatRepo(db.DB)
	message_repo := repos.NewMessageRepo(db.DB)

	auth_service := services.NewAuthSerivce(user_repo, chat_repo, userchat_repo, message_repo)

	auth_controller := controllers.NewAuthController(auth_service)
	user_controller := controllers.NewUserController(user_repo)
	chat_controller := controllers.NewChatController(chat_repo)
	message_controller := controllers.NewMessageController(message_repo)
	signaling_controller := controllers.NewSignalingController()

	err := wails.Run(&options.App{
		Width:     700,
		Height:    512,
		MinWidth:  400,
		MinHeight: 250,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		CSSDragProperty: "--wails-draggable",
		CSSDragValue:    "drag",
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Sept",
				Message: "© 2024 Piheta",
			},
			Preferences: &mac.Preferences{
				TabFocusesLinks:        mac.Enabled,
				TextInteractionEnabled: mac.Enabled,
				FullscreenEnabled:      mac.Disabled,
			},
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup: func(ctx context.Context) {
			auth_controller.SetContext(ctx)
			auth_service.SetContext(ctx)
			auth_service.LogInWithExistingJwt()
		},
		Bind: []interface{}{
			auth_controller,
			user_controller,
			message_controller,
			chat_controller,
			signaling_controller,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}
}
