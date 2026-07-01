// @ai-modified 2026-07-02 wire sessions, CSRF, auth middleware and user routes
package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"mallstock/internal/config"
	"mallstock/internal/database"
	"mallstock/internal/handlers"
	"mallstock/internal/middleware"
	"mallstock/internal/models"
	"mallstock/internal/repository"
	"mallstock/internal/service"
	"mallstock/internal/templates"
	"mallstock/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logLevel := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		logLevel = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Info("database connected")

	tmpl, err := templates.New(web.Templates)
	if err != nil {
		return err
	}

	// Sessions: server-side store in Postgres.
	session := scs.New()
	session.Store = pgxstore.New(pool)
	session.Lifetime = 12 * time.Hour
	session.Cookie.HttpOnly = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = !cfg.IsDev()

	// Repositories.
	userRepo := &repository.UserRepo{DB: pool}
	storeRepo := &repository.StoreRepo{DB: pool}
	categoryRepo := &repository.CategoryRepo{DB: pool}
	supplierRepo := &repository.SupplierRepo{DB: pool}
	productRepo := &repository.ProductRepo{DB: pool}
	movementRepo := &repository.MovementRepo{DB: pool}
	reportRepo := &repository.ReportRepo{DB: pool}

	// Services.
	authSvc := &service.AuthService{Users: userRepo}
	userSvc := &service.UserService{Users: userRepo}
	storeSvc := &service.StoreService{Stores: storeRepo}
	categorySvc := &service.CategoryService{Categories: categoryRepo}
	supplierSvc := &service.SupplierService{Suppliers: supplierRepo}
	productSvc := &service.ProductService{Products: productRepo}
	stockSvc := &service.StockService{Pool: pool, Movements: movementRepo}
	reportSvc := &service.ReportService{Reports: reportRepo, Movements: movementRepo}

	// Handlers.
	base := &handlers.Base{Tmpl: tmpl, Log: log, Session: session}
	dashH := &handlers.DashboardHandler{Base: base, Reports: reportSvc, Stock: stockSvc}
	reportsH := &handlers.ReportsHandler{Base: base, Reports: reportSvc, Stores: storeSvc}
	authH := &handlers.AuthHandler{Base: base, Auth: authSvc}
	usersH := &handlers.UsersHandler{Base: base, Users: userSvc, Stores: storeRepo}
	storesH := &handlers.StoresHandler{Base: base, Stores: storeSvc}
	categoriesH := &handlers.CategoriesHandler{Base: base, Categories: categorySvc}
	suppliersH := &handlers.SuppliersHandler{Base: base, Suppliers: supplierSvc}
	productsH := &handlers.ProductsHandler{
		Base: base, Products: productSvc, Stores: storeSvc,
		Categories: categorySvc, Suppliers: supplierSvc, Stock: stockSvc,
	}
	stockH := &handlers.StockHandler{Base: base, Stock: stockSvc, Products: productSvc, Stores: storeSvc}

	authMW := &middleware.Auth{Session: session, Users: userSvc}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(requestLogger(log))
	r.Use(chimw.Recoverer)

	// Unauthenticated, session-free endpoints.
	r.Get("/healthz", handlers.Healthz(pool))
	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		return err
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Everything else runs inside session + CSRF + user loading.
	r.Group(func(r chi.Router) {
		r.Use(sessionMiddleware(session))
		r.Use(middleware.CSRF(!cfg.IsDev()))
		r.Use(authMW.LoadUser)

		r.Get("/login", authH.LoginForm)
		r.Post("/login", authH.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAuth)

			r.Get("/", dashH.Index)
			r.Post("/logout", authH.Logout)

			// Reports (all roles may view + export).
			r.Route("/reports", func(r chi.Router) {
				r.Get("/", reportsH.Index)
				r.Get("/low-stock", reportsH.LowStock)
				r.Get("/valuation", reportsH.Valuation)
				r.Get("/movements", reportsH.Movements)
				r.Get("/{name}/export.csv", reportsH.ExportCSV)
			})

			manage := middleware.RequireRole(models.RoleAdmin, models.RoleManager)

			// Products: viewing for all, editing for admin/manager.
			r.Route("/products", func(r chi.Router) {
				r.Get("/", productsH.List)
				r.Group(func(r chi.Router) {
					r.Use(manage)
					r.Get("/new", productsH.NewForm)
					r.Post("/", productsH.Create)
					r.Get("/{id}/edit", productsH.EditForm)
					r.Post("/{id}", productsH.Update)
					r.Post("/{id}/delete", productsH.Deactivate)
				})
				r.Get("/{id}", productsH.Detail)

				// Stock operations: every role may record them (plan.md §8).
				r.Get("/{id}/stock-in", stockH.StockForm(models.MovementIn))
				r.Post("/{id}/stock-in", stockH.StockIn)
				r.Get("/{id}/stock-out", stockH.StockForm(models.MovementOut))
				r.Post("/{id}/stock-out", stockH.StockOut)
				r.Post("/{id}/adjust", stockH.Adjust)
			})

			// Movement history.
			r.Get("/movements", stockH.History)

			// Stores.
			r.Route("/stores", func(r chi.Router) {
				r.Get("/", storesH.List)
				r.Group(func(r chi.Router) {
					r.Use(manage)
					r.Get("/new", storesH.NewForm)
					r.Post("/", storesH.Create)
					r.Get("/{id}/edit", storesH.EditForm)
					r.Post("/{id}", storesH.Update)
					r.Post("/{id}/deactivate", storesH.SetActive(false))
					r.Post("/{id}/activate", storesH.SetActive(true))
				})
			})

			// Categories.
			r.Route("/categories", func(r chi.Router) {
				r.Get("/", categoriesH.List)
				r.Group(func(r chi.Router) {
					r.Use(manage)
					r.Get("/new", categoriesH.NewForm)
					r.Post("/", categoriesH.Create)
					r.Get("/{id}/edit", categoriesH.EditForm)
					r.Post("/{id}", categoriesH.Update)
					r.Post("/{id}/delete", categoriesH.Delete)
				})
			})

			// Suppliers.
			r.Route("/suppliers", func(r chi.Router) {
				r.Get("/", suppliersH.List)
				r.Group(func(r chi.Router) {
					r.Use(manage)
					r.Get("/new", suppliersH.NewForm)
					r.Post("/", suppliersH.Create)
					r.Get("/{id}/edit", suppliersH.EditForm)
					r.Post("/{id}", suppliersH.Update)
					r.Post("/{id}/delete", suppliersH.Delete)
				})
			})

			// Admin-only user management.
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireRole(models.RoleAdmin))
				r.Get("/", usersH.List)
				r.Get("/new", usersH.NewForm)
				r.Post("/", usersH.Create)
				r.Get("/{id}/edit", usersH.EditForm)
				r.Post("/{id}", usersH.Update)
				r.Post("/{id}/deactivate", usersH.Deactivate)
				r.Post("/{id}/activate", usersH.Activate)
			})
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func sessionMiddleware(s *scs.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return s.LoadAndSave(next)
	}
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration", time.Since(start).String(),
			)
		})
	}
}
