package main

import (
	"embed"
	"errors"
	"github.com/gin-gonic/gin"
	"html/template"
	"log/slog"
	"net/http"
	"os/exec"
)

func isRunning() (bool, error) {
	cmd := exec.Command("pgrep", "sing-box")
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if _, ok := err.(*exec.ExitError); ok {
		return false, nil
	}
	slog.Warn("Cant check status: ", "error", err)
	return false, err
}

func startService() error {
	cmd := exec.Command("/etc/init.d/singbox", "start")
	if err := cmd.Run(); err != nil {
		slog.Warn("Cant start service...", "error", err)
		return err
	}
	return nil
}

func stopService() error {
	cmd := exec.Command("/etc/init.d/singbox", "stop")
	if err := cmd.Run(); err != nil {
		slog.Warn("Cant stop service...", "error", err)
		return err
	}
	return nil
}

type PostData struct {
	Action string `form:"action"`
}

//go:embed static static/*
var embdfs embed.FS

func main() {
	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	err := router.SetTrustedProxies(nil)
	if err != nil {
		slog.Error("Cant set trusted proxies: %s", err)
		panic(err)
	}

	// http.FS - просто обёртка
	router.StaticFileFS("/favicon.png", "static/favicon.png", http.FS(embdfs))
	// It is so inconvenient...
	// Why not router.LoadHTMLFilesFs(fs, path)?
	router.SetHTMLTemplate(template.Must(template.New("meow").ParseFS(embdfs, "static/*.gohtml")))

	//router.StaticFile("/favicon.png", "./static/favicon.png")
	//router.LoadHTMLFiles("./static/index.gohtml")

	indexPageHandler := func(ctx *gin.Context, err error) {
		status, e := isRunning()
		ctx.HTML(http.StatusOK, "index.gohtml", gin.H{
			"running":       status,
			"error_message": errors.Join(e, err),
		})
	}

	setErrorCookie := func(ctx *gin.Context, err error) {
		ctx.SetCookie("Error", err.Error(), 2, "/", "", false, true)
	}

	deleteErrorCookie := func(ctx *gin.Context) {
		ctx.SetCookie("Error", "Haha, nothing there", -1, "/", "", false, true)
	}

	router.GET("/", func(ctx *gin.Context) {
		cookie, e := ctx.Cookie("Error")
		if e != nil {
			indexPageHandler(ctx, nil)
		} else {
			deleteErrorCookie(ctx)
			indexPageHandler(ctx, errors.New(cookie))
		}
	})

	router.POST("/form", func(ctx *gin.Context) {
		data := &PostData{}
		err := ctx.Bind(data)
		if err != nil {
			slog.Info("Cannot parse POST form data: %s", err)
			ctx.String(http.StatusBadRequest, "Cannot parse POST form data: %s", err)
			return
		}
		if data.Action != "start" && data.Action != "stop" {
			slog.Info("Wrong action in POST form data: %s", data.Action)
			ctx.String(http.StatusBadRequest, "Wrong action in POST form data: %s", data.Action)
			return
		}
		var action func() error
		if data.Action == "start" {
			action = startService
		} else {
			action = stopService
		}
		if e := action(); e != nil {
			setErrorCookie(ctx, e)
		}
		ctx.Redirect(http.StatusFound, "/")
	})

	err = router.Run("0.0.0.0:2024")
	if err != nil {
		slog.Error("Server down with ", "error", err)
		panic(err)
	}
}
