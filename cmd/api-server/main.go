package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/WenDoge/rest-api/docs"
	"github.com/WenDoge/rest-api/internal/config"
	"github.com/WenDoge/rest-api/internal/database"
	"github.com/WenDoge/rest-api/internal/handlers"
	"github.com/WenDoge/rest-api/internal/middleware"
	"github.com/WenDoge/rest-api/internal/repository"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

//	@title			Users API
//	@version		1.0
//	@description	регистрация, логин, профиль
//	@host			localhost:8080
//	@BasePath		/api/v1

//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Формат: "Bearer <token>"

var (
	_ handlers.UserRepository  = (*repository.UserRepository)(nil)
	_ handlers.TokenRepository = (*repository.TokenRepository)(nil)
)

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config", err)
	}

	pool, err := database.Connect(config.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database", err)
	}

	router := gin.Default()
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api/v1")

	api.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "REST API is running",
		})
	})

	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewTokenRepository(pool)

	userHandler := handlers.NewUserHandler(userRepo, config)
	authHandler := handlers.NewAuthHandler(userRepo, tokenRepo, config)

	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/logout", authHandler.Logout)

	usersGroup := api.Group("/users")
	usersGroup.Use(middleware.AuthMiddleware(config))
	usersGroup.GET("", userHandler.List)
	usersGroup.GET("/my", userHandler.GetMe)
	usersGroup.PATCH("/my", userHandler.Update)
	usersGroup.DELETE("/my", userHandler.Delete)

	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	serverErr := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	log.Println("Server started on port:", config.Port)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	exitCode := 0

	select {
	case sig := <-signalChan:
		log.Printf("Received signal %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
		exitCode = 1
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
		exitCode = 1
	}
	log.Println("Server stopped")

	pool.Close()
	log.Println("Database pool closed")

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
