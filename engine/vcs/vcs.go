package vcs

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

// ApplyConfiguration apply an object of type hooks.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}
	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}

	return nil
}

func (s *Service) getConsumer(name string) (sdk.VCSServer, error) {
	serverCfg := s.Cfg.Servers[name]
	if serverCfg.Github != nil {
		return github.New(serverCfg.Github.ClientID, serverCfg.Github.ClientSecret, s.Cache), nil
	}
	return nil, sdk.ErrNotFound
}

// Serve will start the http api server
func (s *Service) Serve(c context.Context) error {
	if s.Cfg.Name == "" {
		s.Cfg.Name = hatchery.GenerateName("vcs", "")
	}

	ctx, cancel := context.WithCancel(c)
	defer cancel()

	log.Info("VCS> Starting service %s...", s.Cfg.Name)

	//Instanciate a cds client
	s.cds = cdsclient.NewService(s.Cfg.API.HTTP.URL)

	//First register(heartbeat)
	if err := s.doHeartbeat(); err != nil {
		log.Error("VCS> Unable to register: %v", err)
		return err
	}
	log.Info("VCS> Service registered")

	//Start the heartbeat gorourine
	go func() {
		if err := s.heartbeat(ctx); err != nil {
			log.Error("%v", err)
			cancel()
		}
	}()

	//Init the cache
	var errCache error
	s.Cache, errCache = cache.New(s.Cfg.Cache.Redis.Host, s.Cfg.Cache.Redis.Password, s.Cfg.Cache.TTL)
	if errCache != nil {
		return errCache
	}

	//Init the http server
	s.initRouter(ctx)
	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	//Gracefully shutdown the http server
	go func() {
		select {
		case <-ctx.Done():
			log.Info("VCS> Shutdown HTTP Server")
			server.Shutdown(ctx)
		}
	}()

	//Start the http server
	log.Info("VCS> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("VCS> Cannot start cds-hooks: %s", err)
	}

	return ctx.Err()
}
