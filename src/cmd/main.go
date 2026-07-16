package cmd

import (
	cmd "blog_api/src/cmd/router"
	"blog_api/src/config"
	"blog_api/src/repositories"
	friendsRepositories "blog_api/src/repositories/friend"
	"blog_api/src/service"
	botService "blog_api/src/service/bot"
	"blog_api/src/service/oss"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Run 启动应用程序
func Run() {
	startTime := time.Now()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[main]加载配置失败: %v", err)
	}
	db, err := repositories.InitDB(cfg)
	if err != nil {
		log.Fatalf("[main]初始化数据库失败: %v", err)
	}
	if err := friendsRepositories.InsertFriendLinks(db, cfg.FriendLinks); err != nil {
		log.Printf("[main]无法插入友链: %v", err)
	}
	if err := service.ScanAndSaveImages(db); err != nil {
		log.Printf("[main]无法扫描和保存图片: %v", err)
	}
	if err := oss.ValidateOSSConfig(); err != nil {
		log.Printf("[main][OSS]配置校验失败: %v", err)
	}
	router := cmd.SetupRouter(db, cfg, startTime)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.ListenAddress, cfg.Port)
		log.Printf("[main][Http]HTTP 服务器启动于 %s", addr)
		server := &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       2 * time.Minute,
			WriteTimeout:      2 * time.Minute,
			IdleTimeout:       time.Minute,
		}
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("[main][Http]启动 HTTP 服务器失败: %v", err)
		}
	}()

	botService.StartListeners(db, cfg)
	StartCronJobs(db)
	log.Println("[main][App]应用程序启动成功。HTTP 服务器和 cron 任务正在运行。")

	select {}
}
