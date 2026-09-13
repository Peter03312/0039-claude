// Command api 提供管风琴联动核查的 HTTP 接口。
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"organverify/api/internal/core"
)

type snapshotRequest struct {
	Name   string             `json:"name"`
	Input  core.VerifyRequest `json:"input"`
	Result core.Result        `json:"result"`
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	store := core.NewSnapshotStore(os.Getenv("SNAPSHOT_FILE"))

	api := r.Group("/api")
	api.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api.POST("/verify", func(c *gin.Context) {
		var req core.VerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errors": []core.FieldError{
				{Path: "", Message: "请求体不是合法 JSON：" + err.Error()},
			}})
			return
		}
		res, errs := core.Verify(req)
		if len(errs) > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": errs})
			return
		}
		c.JSON(http.StatusOK, res)
	})

	api.POST("/snapshots", func(c *gin.Context) {
		var req snapshotRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"errors": []core.FieldError{
				{Path: "", Message: "请求体不是合法 JSON：" + err.Error()},
			}})
			return
		}
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = "未命名快照"
		}
		snap := store.Add(name, req.Input, req.Result)
		c.JSON(http.StatusCreated, snap)
	})

	api.GET("/snapshots", func(c *gin.Context) {
		c.JSON(http.StatusOK, store.List())
	})

	api.GET("/snapshots/:id", func(c *gin.Context) {
		snap, ok := store.Get(c.Param("id"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"errors": []core.FieldError{
				{Path: "id", Message: "快照不存在"},
			}})
			return
		}
		c.JSON(http.StatusOK, snap)
	})

	api.DELETE("/snapshots/:id", func(c *gin.Context) {
		if !store.Delete(c.Param("id")) {
			c.JSON(http.StatusNotFound, gin.H{"errors": []core.FieldError{
				{Path: "id", Message: "快照不存在"},
			}})
			return
		}
		c.Status(http.StatusNoContent)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("api listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
