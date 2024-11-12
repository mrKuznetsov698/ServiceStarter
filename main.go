package main

import (
	"errors"
	"github.com/gin-gonic/gin"
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

func main() {
	router := gin.Default()

	err := router.SetTrustedProxies(nil)
	if err != nil {
		slog.Error("Cant set trusted proxies: %s", err)
		panic(err)
	}

	router.StaticFile("/favicon.png", "./static/favicon.png")
	router.LoadHTMLFiles("./static/index.gohtml")

	indexPageHandler := func(ctx *gin.Context, errs []error) {
		status, e := isRunning()
		//if errs == nil {
		//	errs = make([]error, 0)
		//}
		errs = append(errs, e)
		ctx.HTML(http.StatusOK, "index.gohtml", gin.H{
			"running":       status,
			"error_message": errors.Join(errs...),
		})
	}

	router.GET("/", func(ctx *gin.Context) {
		indexPageHandler(ctx, []error{})
	})

	router.POST("/", func(ctx *gin.Context) {
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
		if data.Action == "start" {
			if e := startService(); e != nil {
				indexPageHandler(ctx, []error{e})
			} else {
				indexPageHandler(ctx, []error{})
			}
		} else {
			if e := stopService(); e != nil {
				indexPageHandler(ctx, []error{e})
			} else {
				indexPageHandler(ctx, []error{})
			}
		}
		//ctx.Redirect(http.StatusMovedPermanently, "/")
	})

	router.GET("/start", func(ctx *gin.Context) {
		e := startService()
		if e != nil {
			ctx.String(http.StatusInternalServerError, "Something went wrong while starting: %s", e)
		} else {
			ctx.String(http.StatusOK, "sing-box up & running!")
		}
	})
	router.GET("/stop", func(ctx *gin.Context) {
		e := stopService()
		if e != nil {
			ctx.String(http.StatusInternalServerError, "Something went wrong while stopping: %s", e)
		} else {
			ctx.String(http.StatusOK, "sing-box stopped!")
		}
	})
	router.GET("/running", func(ctx *gin.Context) {
		status, e := isRunning()
		if e != nil {
			ctx.String(http.StatusInternalServerError, "Something went wrong while checking: %s", e)
		} else if status {
			ctx.String(http.StatusOK, "sing-box is running!")
		} else {
			ctx.String(http.StatusOK, "sing-box is stopped")
		}
	})

	err = router.Run("0.0.0.0:2024")
	if err != nil {
		slog.Error("Server down with ", "error", err)
		panic(err)
	}
}
