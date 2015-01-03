package main

import (
	"expvar"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"regexp"
	"time"

	"github.com/codegangsta/cli"
	"github.com/gorilla/mux"
)

// CmdServe is the implementation of the godinstall "serve" command
func CmdServe(c *cli.Context) {
	listenAddress := c.String("listen")

	ttl := c.String("ttl")
	maxReqs := c.Int("max-requests")
	repoBase := c.String("repo-base")
	cookieName := c.String("cookie-name")
	uploadHook := c.String("upload-hook")
	preGenHook := c.String("pre-gen-hook")
	postGenHook := c.String("post-gen-hook")
	poolPattern := c.String("default-pool-pattern")
	verifyChanges := c.Bool("default-verify-changes")
	verifyChangesSufficient := c.Bool("default-verify-changes-sufficient")
	acceptLoneDebs := c.Bool("default-accept-lone-debs")
	verifyDebs := c.Bool("default-verify-debs")
	pruneRulesStr := c.String("default-prune")
	autoTrim := c.Bool("default-auto-trim")
	trimLen := c.Int("default-auto-trim-length")

	if repoBase == "" {
		log.Println("You must pass --repo-base")
		return
	}

	expire, err := time.ParseDuration(ttl)
	if err != nil {
		log.Println(err.Error())
		return
	}

	storeDir := repoBase + "/store"
	tmpDir := repoBase + "/tmp"
	publicDir := repoBase + "/archive"

	_, patherr := os.Stat(publicDir)
	if os.IsNotExist(patherr) {
		err = os.Mkdir(publicDir, 0777)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
	_, patherr = os.Stat(storeDir)
	if os.IsNotExist(patherr) {
		err = os.Mkdir(storeDir, 0777)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}

	_, patherr = os.Stat(tmpDir)
	if os.IsNotExist(patherr) {
		err = os.Mkdir(tmpDir, 0777)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}

	_, err = ParsePruneRules(pruneRulesStr)
	if err != nil {
		log.Println(err.Error())
		return
	}

	_, err = regexp.CompilePOSIX("^(" + poolPattern + ")")
	if err != nil {
		log.Println(err.Error())
		return
	}

	state.Archive = NewAptBlobArchive(
		&storeDir,
		&tmpDir,
		&publicDir,
		ReleaseConfig{
			VerifyChanges:           verifyChanges,
			VerifyChangesSufficient: verifyChangesSufficient,
			VerifyDebs:              verifyDebs,
			AcceptLoneDebs:          acceptLoneDebs,
			PruneRules:              pruneRulesStr,
			AutoTrim:                autoTrim,
			AutoTrimLength:          trimLen,
			PoolPattern:             poolPattern,
		},
	)

	state.UpdateChannel = make(chan UpdateRequest)
	state.SessionManager = NewUploadSessionManager(
		expire,
		&tmpDir,
		state.Archive,
		NewScriptHook(&uploadHook),
		state.UpdateChannel,
	)

	cfg.CookieName = cookieName
	cfg.PreGenHook = NewScriptHook(&preGenHook)
	cfg.PostGenHook = NewScriptHook(&postGenHook)

	state.Lock = NewGovernor(maxReqs)
	state.getCount = expvar.NewInt("GetRequests")

	go updater()

	r := mux.NewRouter()

	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.Handle("/debug/vars", http.DefaultServeMux)

	r.PathPrefix("/repo/").HandlerFunc(makeHTTPDownloadHandler())
	r.PathPrefix("/upload").HandlerFunc(httpUploadHandler)

	r.HandleFunc("/dists", httpDistsHandler)
	r.HandleFunc("/dists/{name}", httpDistsHandler)
	r.HandleFunc("/dists/{name}/config", httpConfigHandler)
	r.HandleFunc("/dists/{name}/config/signingkey", httpConfigSigningKeyHandler)
	r.HandleFunc("/dists/{name}/config/publickeys", httpConfigPublicKeysHandler)
	r.HandleFunc("/dists/{name}/config/publickeys/{id}", httpConfigPublicKeysHandler)
	r.HandleFunc("/dists/{name}/log", httpLogHandler)
	r.HandleFunc("/dists/{name}/upload", httpUploadHandler)
	r.HandleFunc("/dists/{name}/upload/{session}", httpUploadHandler)

	http.ListenAndServe(listenAddress, r)

}