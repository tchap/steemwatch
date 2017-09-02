package server

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/url"
	"strings"

	"github.com/tchap/steemwatch/config"
	"github.com/tchap/steemwatch/server/auth"
	"github.com/tchap/steemwatch/server/auth/facebook"
	"github.com/tchap/steemwatch/server/auth/github"
	"github.com/tchap/steemwatch/server/auth/google"
	"github.com/tchap/steemwatch/server/auth/reddit"
	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/db"
	"github.com/tchap/steemwatch/server/routes/api/eventstream"
	"github.com/tchap/steemwatch/server/routes/api/notifiers/discord"
	"github.com/tchap/steemwatch/server/routes/api/notifiers/slack"
	"github.com/tchap/steemwatch/server/routes/api/notifiers/steemitchat"
	"github.com/tchap/steemwatch/server/routes/api/notifiers/telegram"
	"github.com/tchap/steemwatch/server/routes/api/profile"
	"github.com/tchap/steemwatch/server/routes/api/v1/info"
	"github.com/tchap/steemwatch/server/routes/home"
	"github.com/tchap/steemwatch/server/routes/logout"
	"github.com/tchap/steemwatch/server/sessions"
	"github.com/tchap/steemwatch/server/users/stores/mongodb"
	"github.com/tchap/steemwatch/server/views"

	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/engine/fasthttp"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/tomb.v2"
)

type Context struct {
	EventStreamManager *eventstream.Manager

	listener net.Listener

	discordSession *discordgo.Session

	t tomb.Tomb
}

func Run(mongo *mgo.Database, cfg *config.Config) (*Context, *discordgo.Session, error) {
	serverCtx := &context.Context{}

	// Environment.
	switch cfg.Env {
	case "development":
		serverCtx.Env = context.EnvironmentDevelopment
	case "production":
		serverCtx.Env = context.EnvironmentProduction
		serverCtx.SSLEnabled = true
	default:
		return nil, nil, errors.New("invalid environment: " + cfg.Env)
	}

	// Database.
	serverCtx.DB = mongo

	// User store.
	userStore := mongodb.NewUserStore(mongo.C("users"))

	// Session manager.
	hashKey, blockKey, err := getSecureCookieKeys(mongo)
	if err != nil {
		return nil, nil, err
	}
	sessionManager, err := sessions.NewSessionManager(hashKey, blockKey, userStore)
	if err != nil {
		return nil, nil, err
	}

	serverCtx.SessionManager = sessionManager

	// Server context.
	canonicalURL, err := url.Parse(cfg.CanonicalURL)
	if err != nil {
		return nil, nil, errors.Wrap(err, "invalid canonical URL")
	}

	serverCtx.CanonicalURL = canonicalURL

	// Echo.
	e := echo.New()

	// Templates.
	renderer, err := views.NewRenderer("./server/views/*.html")
	if err != nil {
		return nil, nil, err
	}
	e.SetRenderer(renderer)

	// Assets.
	e.Static("/app", "server/app")
	e.Static("/modules", "server/app/node_modules")
	e.Static("/assets/css", "server/assets/css")
	e.Static("/assets/js", "server/assets/js")
	e.Static("/assets/img", "server/assets/img")
	e.Static("/assets/fonts", "server/assets/fonts")
	e.Static("/assets/bootstrap", "server/app/node_modules/bootstrap/dist")

	// Middleware
	e.Pre(middleware.AddTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())

	csrf := middleware.CSRF([]byte("secret"))

	// Web
	homeHandler := home.NewHandlerFunc(serverCtx)

	e.GET("/", homeHandler, csrf)
	e.GET("/logout/", logout.NewHandlerFunc(serverCtx), csrf)

	e.GET("/home/", homeHandler, csrf)
	e.GET("/events/", homeHandler, csrf)
	e.GET("/eventstream/", homeHandler, csrf)
	e.GET("/notifications/", homeHandler, csrf)
	e.GET("/profile/", homeHandler, csrf)

	facebookCallbackPath, _ := url.Parse("/auth/facebook/callback")
	facebookCallback := serverCtx.CanonicalURL.ResolveReference(facebookCallbackPath).String()
	facebookAuth := facebook.NewAuthenticator(
		cfg.FacebookClientId, cfg.FacebookClientSecret, facebookCallback)
	auth.Bind(serverCtx, e.Group("/auth/facebook", csrf), facebookAuth)

	redditCallbackPath, _ := url.Parse("/auth/reddit/callback")
	redditCallback := serverCtx.CanonicalURL.ResolveReference(redditCallbackPath).String()
	redditAuth := reddit.NewAuthenticator(
		cfg.RedditClientId, cfg.RedditClientSecret, redditCallback, serverCtx.SSLEnabled)
	auth.Bind(serverCtx, e.Group("/auth/reddit", csrf), redditAuth)

	googleCallbackPath, _ := url.Parse("/auth/google/callback")
	googleCallback := serverCtx.CanonicalURL.ResolveReference(googleCallbackPath).String()
	googleAuth := google.NewAuthenticator(
		cfg.GoogleClientId, cfg.GoogleClientSecret, googleCallback)
	auth.Bind(serverCtx, e.Group("/auth/google", csrf), googleAuth)

	githubCallbackPath, _ := url.Parse("/auth/github/callback")
	githubCallback := serverCtx.CanonicalURL.ResolveReference(githubCallbackPath).String()
	githubAuth := github.NewAuthenticator(
		cfg.GitHubClientId, cfg.GitHubClientSecret, githubCallback)
	auth.Bind(serverCtx, e.Group("/auth/github", csrf), githubAuth)

	// Public API
	info.Bind(serverCtx, e.Group("/api/v1/info"))

	// API
	api := e.Group("/api", csrf, auth.Required(serverCtx))

	// API - Events
	db.BindList(serverCtx, api.Group("/events/:kind/:list"))

	// API - Event Stream
	manager := eventstream.NewManager()
	manager.Bind(serverCtx, api.Group("/eventstream"))

	// API - Notifiers
	slack.Bind(serverCtx, api.Group("/notifiers/slack"))
	steemitchat.Bind(serverCtx, api.Group("/notifiers/steemit-chat"))

	// Telegram
	botSecret := make([]byte, 256/8)
	if _, err := rand.Read(botSecret); err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate Telegram bot secret")
	}
	botSecretHex := hex.EncodeToString(botSecret)

	botURL, _ := url.Parse("/bots/telegram/both-" + botSecretHex)
	botWebhookURL := serverCtx.CanonicalURL.ResolveReference(botURL)

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize Telegram bot")
	}

	if !strings.Contains(cfg.CanonicalURL, "localhost") {
		if _, err := bot.SetWebhook(tgbotapi.NewWebhook(botWebhookURL.String())); err != nil {
			return nil, nil, errors.Wrap(err, "failed to register Telegram webhook URL")
		}
	}

	telegram.BindWebhook(serverCtx, e.Group(botURL.String()))
	telegram.BindAPI(serverCtx, api.Group("/notifiers/telegram"))

	// API - Profile
	profile.Bind(serverCtx, api.Group("/profile"))

	// Start server
	listener, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to start the web server")
	}

	ctx := &Context{
		EventStreamManager: manager,
		listener:           listener,
	}

	// Discord
	dg, err := discord.InitBot(&ctx.t, cfg.DiscordBotToken, serverCtx)
	if err != nil {
		return nil, nil, err
	}

	discord.BindAPI(serverCtx, api.Group("/notifiers/discord"))

	// Start listening.
	ctx.t.Go(func() error {
		e.Run(fasthttp.WithConfig(engine.Config{
			Listener: listener,
		}))
		return nil
	})

	go func() {
		<-ctx.t.Dying()
		listener.Close()
	}()

	return ctx, dg, nil
}

func (ctx *Context) Interrupt() {
	ctx.t.Kill(nil)
}

func (ctx *Context) Wait() error {
	return ctx.t.Wait()
}
