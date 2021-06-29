package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/judwhite/go-svc"
	"github.com/lim-team/LiMaoIM/internal/lim"
	"github.com/tangtaoit/limnet/pkg/limlog"
)

type program struct {
	l *lim.LiMao
}

func main() {
	prg := &program{}

	if err := svc.Run(prg); err != nil {
		log.Fatal(err)
	}
}

func (p *program) Init(env svc.Environment) error {
	if env.IsWindowsService() {
		dir := filepath.Dir(os.Args[0])
		return os.Chdir(dir)
	}
	return nil
}

type environments []string

func (e *environments) String() string {
	return "environments string"
}

func (e *environments) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func (p *program) Start() error {

	// ########## LiMaoIM ##########
	configPtr := flag.String("c", "", "Configuration file")
	var envList environments
	flag.Var(&envList, "e", "Environment variables can override configuration")
	flag.Parse()

	p.setEnv(envList)

	// ########## Init Config ##########
	opts := lim.NewOptions()
	if *configPtr != "" {
		opts.Load(*configPtr)
	} else {
		opts.Load()
	}
	gin.SetMode(opts.Mode)
	if opts.Mode == lim.DebugMode {
		os.Setenv("CONFIGOR_DEBUG_MODE", "true")
	}

	// ########## LiMaoIM Start ##########
	p.l = lim.New(opts)
	return p.l.Start(nil)
}

func (p *program) setEnv(envList environments) {
	if len(envList) > 0 {
		for _, env := range envList {
			keyValues := strings.Split(env, "=")
			key := keyValues[0]
			value := keyValues[1]
			os.Setenv(key, value)
		}
	}
}

func (p *program) Stop() error {
	limlog.Debug("LiMao退出")
	if p.l != nil {
		if err := p.l.Stop(); err != nil {
			return err
		}
	}

	return nil
}
