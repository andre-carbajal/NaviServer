package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"naviger/internal/api"
	"naviger/internal/backup"
	"naviger/internal/config"
	"naviger/internal/jvm"
	"naviger/internal/runner"
	"naviger/internal/server"
	"naviger/internal/storage"
	"naviger/internal/updater"
	"naviger/internal/ws"

	"github.com/emersion/go-autostart"
	"github.com/getlantern/systray"
	"github.com/pkg/browser"
)

//go:embed icon.png
var iconPngData []byte

//go:embed icon.ico
var iconIcoData []byte

var headless bool

func main() {
	flag.BoolVar(&headless, "headless", false, "Run in headless mode (no GUI)")
	flag.Parse()

	if headless {
		runHeadless()
	} else {
		runDesktop()
	}
}

func runDesktop() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTooltip("Naviger Daemon")
	if runtime.GOOS == "windows" {
		systray.SetIcon(iconIcoData)
	} else {
		systray.SetIcon(iconPngData)
	}

	mStatus := systray.AddMenuItem("Status: Running", "Current status")
	mStatus.Disable()
	systray.AddSeparator()
	mOpenUI := systray.AddMenuItem("Open Web UI", "Open dashboard")
	mRestart := systray.AddMenuItem("Restart Daemon", "Reload configuration and restart server")
	mStartLogin := systray.AddMenuItemCheckbox("Start at Login", "Run on startup", false)
	systray.AddSeparator()

	mVersion := systray.AddMenuItem(fmt.Sprintf("Version: %s", updater.CurrentVersion), "Current version")
	mVersion.Disable()

	go func() {
		info, err := updater.CheckForUpdates()
		if err == nil && info.UpdateAvailable {
			mVersion.SetTitle(fmt.Sprintf("Update Available: %s ⚠️", info.LatestVersion))
			mVersion.SetTooltip("Click to open release page")
			mVersion.Enable()
		}
	}()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Exit", "Quit application")

	executable, err := os.Executable()
	var appAutoStart *autostart.App
	if err == nil {
		appAutoStart = &autostart.App{
			Name:        "Naviger",
			DisplayName: "Naviger Daemon",
			Exec:        []string{executable},
		}
		if appAutoStart.IsEnabled() {
			mStartLogin.Check()
		}
	} else {
		log.Printf("Error getting executable: %v", err)
		mStartLogin.Disable()
	}

	var ctx context.Context
	var cancel context.CancelFunc
	var wg sync.WaitGroup

	startService := func() {
		ctx, cancel = context.WithCancel(context.Background())
		wg.Add(1)
		go func() {
			defer wg.Done()
			startDaemonService(ctx)
		}()
		mStatus.SetTitle("Status: Running")
	}

	startService()

	go func() {
		for {
			select {
			case <-mOpenUI.ClickedCh:
				port := config.GetPort()
				_ = browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))

			case <-mRestart.ClickedCh:
				mStatus.SetTitle("Status: Restarting...")
				log.Println("Reiniciando servicio...")

				cancel()
				wg.Wait()

				startService()
				log.Println("Servicio reiniciado.")

			case <-mStartLogin.ClickedCh:
				if appAutoStart == nil {
					continue
				}
				if mStartLogin.Checked() {
					if err := appAutoStart.Disable(); err == nil {
						mStartLogin.Uncheck()
					}
				} else {
					if err := appAutoStart.Enable(); err == nil {
						mStartLogin.Check()
					}
				}

			case <-mVersion.ClickedCh:
				info, err := updater.CheckForUpdates()
				if err == nil && info.UpdateAvailable {
					_ = browser.OpenURL(info.ReleaseURL)
				}

			case <-mQuit.ClickedCh:
				mStatus.SetTitle("Status: Stopping...")
				cancel()
				wg.Wait()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	log.Println("Application exited.")
}

func runHeadless() {
	log.Println("Running in headless mode...")

	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go startDaemonService(ctx)

	<-sigs
	log.Println("Signal received, shutting down...")
	cancel()

	time.Sleep(1 * time.Second)
}

func startDaemonService(ctx context.Context) {
	fmt.Println("Starting Naviger Daemon...")

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("Error getting config dir: %v", err)
		return
	}

	appName := "naviger"
	if config.IsDev() {
		appName = "naviger-dev"
	}
	configDir := filepath.Join(userConfigDir, appName)

	cfg, err := config.LoadConfig(configDir)
	if err != nil {
		log.Printf("Error loading config: %v", err)
		return
	}

	for _, path := range []string{cfg.ServersPath, cfg.BackupsPath, cfg.RuntimesPath} {
		_ = os.MkdirAll(path, 0755)
	}

	store, err := storage.NewGormStore(cfg.DatabasePath)
	if err != nil {
		log.Printf("Fatal DB Error: %v", err)
		return
	}

	jvmMgr := jvm.NewManager(cfg.RuntimesPath)
	srvMgr := server.NewManager(cfg.ServersPath, store)
	bufferSize := config.DefaultLogBufferSize
	if val, err := store.GetSetting("log_buffer_size"); err == nil {
		if n, err := strconv.Atoi(val); err == nil && n >= 0 {
			bufferSize = n
		}
	}
	hubManager := ws.NewHubManager(bufferSize)
	supervisor := runner.NewSupervisor(store, jvmMgr, hubManager, cfg.ServersPath)
	backupManager := backup.NewManager(cfg.ServersPath, cfg.BackupsPath, store)

	if err := supervisor.ResetRunningStates(); err != nil {
		log.Printf("Warning resetting states: %v", err)
	}

	apiServer := api.NewAPIServer(srvMgr, supervisor, store, hubManager, backupManager, cfg)
	listenAddr := fmt.Sprintf(":%d", config.GetPort())

	httpServer := apiServer.CreateHTTPServer(listenAddr)

	go func() {
		log.Printf("API Listening on %s", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP Server Error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutting down HTTP server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP Shutdown error: %v", err)
	}

	log.Println("Daemon stopped cleanly.")
}
