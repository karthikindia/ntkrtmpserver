package main
// hlswatch - keep track of hls viewer stats
// Copyright (C) 2017 Maximilian Pachl

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// --------------------------------------------------------------------------------------
//  imports
// --------------------------------------------------------------------------------------

import (
    "log"
    "runtime"
    "os"
    "os/signal"
    "net/http"
    "syscall"
    "context"
    "time"

    "github.com/faryon93/hlswatch/handler"
    "github.com/faryon93/hlswatch/state"
    "github.com/faryon93/hlswatch/config"
)


// --------------------------------------------------------------------------------------
//  constants
// --------------------------------------------------------------------------------------

const (
    SHUTDOWN_GRACEPERIOD = 5 * time.Second
)


// --------------------------------------------------------------------------------------
//  global variables
// --------------------------------------------------------------------------------------

var (
    // configuration options
    configFile = "/etc/hlswatch.conf"

    // runtime variables
    Ctx *state.State = state.New()
)


// --------------------------------------------------------------------------------------
//  application entry
// --------------------------------------------------------------------------------------

func main() {
    log.Println(GetAppIdentifier())

    // setup go environment to use all available cpu cores
    runtime.GOMAXPROCS(runtime.NumCPU())

    // parse command line arguments
    if len(os.Args) > 1 {
        configFile = os.Args[1]
    }

    // load and parse the configuration file
    conf, err := config.Load(configFile)
    if err != nil {
        log.Println("failed to load configuration file:", err.Error())
        os.Exit(-1)
    }
    Ctx.Conf = conf

    // setup the http static file server serving the playlists
    // TODO: gzip compression for playlist, caching in ram
    rootfs := http.Dir(conf.Common.HlsPath)
    mux := http.NewServeMux()
    mux.Handle("/", handler.Hls(Ctx, http.FileServer(rootfs)))
    route(mux, "/stats", handler.Stats, Ctx)
    srv := &http.Server{Addr: conf.Common.Listen, Handler: mux}

    // serve the content via http
    go func() {
        var err error = nil

        // setup a tls server if configured
        if conf.IsSslEnabled() {
            err = srv.ListenAndServeTLS(conf.Common.SslCertificate,
                                        conf.Common.SslPrivateKey)

        // plain old http server
        } else {
            err = srv.ListenAndServe()
        }

        if err != nil {
            log.Println("failed start http server:", err.Error())
            Ctx.Shutdown() // gracefull shutdown the application
        }
    }()
    log.Println("http is listening on", conf.Common.Listen)

    // fire the statistics computation task
    // TODO: close connections, ...
    go InfluxMetrics(Ctx)
    go StreamWatcher(Ctx)

    // wait for a signal to shutdown the application
    wait(os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
    log.Println("gracefully shutting down application...")

    // gracefully shutdown the http server
    ctx, _ := context.WithTimeout(context.Background(), SHUTDOWN_GRACEPERIOD)
    srv.Shutdown(ctx)

    log.Println("application successfully exited")
}


// --------------------------------------------------------------------------------------
//  helper functions
// --------------------------------------------------------------------------------------

func wait(sig ...os.Signal) {
    signals := make(chan os.Signal)
    signal.Notify(signals, sig...)

    // wait for an OS signal or a signal by the close Channel
    select {
        case <- signals:
        case <- Ctx.CloseChan:
    }
}

func route(mux *http.ServeMux, pattern string, handler handler.Handler, ctx *state.State) {
    f := http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
        handler(ctx, w, r)
    })
    mux.Handle(pattern, f)
}
