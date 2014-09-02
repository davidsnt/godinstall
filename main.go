package main

// Package GoDInstall implements a web service for serving, and manipulating
// debian Apt repositories. The original motivation was to provide a synchronous
// interface for package upload. A package is available for download from the
// repository at the point when the server confirms the package has been
// uploaded.
//   It is primarily aimed at use in continuous delivery processes.

import (
	"flag"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"code.google.com/p/go.crypto/openpgp"
)

// HTTP handler for the server /
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Nothing to see here"))
}

// Looks an email address up in a pgp keyring
func getKeyByEmail(keyring openpgp.EntityList, email string) *openpgp.Entity {
	for _, entity := range keyring {
		for _, ident := range entity.Identities {
			if ident.UserId.Email == email {
				return entity
			}
		}
	}

	return nil
}

func main() {
	// Setup CLI flags
	listenAddress := flag.String("listen", ":3000", "ip:port to listen on")
	ttl := flag.String("ttl", "60s", "Session life time")
	maxReqs := flag.Int("max-requests", 4, "Maximum concurrent requests")
	repoBase := flag.String("repo-base", "/tmp/myrepo", "Location of repository root")
	tmpDir := flag.String("tmp-dir", "/tmp/up", "Location for temporary storage")
	storeDir := flag.String("store-dir", "/tmp/store", "Location for persitant storage")
	cookieName := flag.String("cookie-name", "godinstall-sess", "Name for the sessio ookie")
	uploadHook := flag.String("upload-hook", "", "Script to run after for each uploaded file")
	preAftpHook := flag.String("pre-aftp-hook", "", "Script to run before apt-ftparchive")
	postAftpHook := flag.String("post-aftp-hook", "", "Script to run after apt-ftparchive")
	poolPattern := flag.String("pool-pattern", "[a-z]|lib[a-z]", "A pattern to match package prefixes to split into directories in the pool")
	validateChanges := flag.Bool("validate-changes", true, "Validate signatures on changes files")
	validateChangesSufficient := flag.Bool("validate-changes-sufficient", true, "If we are given a signed chnages file, we wont validate individual debs")
	acceptLoneDebs := flag.Bool("accept-lone-debs", true, "Accept individual debs for upload")
	validateDebs := flag.Bool("validate-debs", true, "Validate signatures on deb files")
	pubringFile := flag.String("gpg-pubring", "", "Public keyring file")
	privringFile := flag.String("gpg-privring", "", "Private keyring file")
	signerEmail := flag.String("signer-email", "", "Key Email to use for signing releases")

	flag.Parse()

	expire, err := time.ParseDuration(*ttl)
	if err != nil {
		log.Println(err.Error())
		return
	}

	poolRegexp, err := regexp.CompilePOSIX("^(" + *poolPattern + ")")

	if err != nil {
		log.Println(err.Error())
		return
	}

	var pubRing openpgp.EntityList
	if *pubringFile != "" {
		pubringReader, err := os.Open(*pubringFile)
		if err != nil {
			log.Println(err.Error())
			return
		}

		pubRing, err = openpgp.ReadKeyRing(pubringReader)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}

	var privRing openpgp.EntityList
	if *privringFile != "" {
		privringReader, err := os.Open(*privringFile)
		if err != nil {
			log.Println(err.Error())
			return
		}

		privRing, err = openpgp.ReadKeyRing(privringReader)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}

	if *validateChanges || *validateDebs {
		if privRing == nil || pubRing == nil {
			log.Println("Validation requested, but keyrings not loaded")
			return
		}
	}

	var signerId *openpgp.Entity
	if *signerEmail != "" {
		signerId = getKeyByEmail(privRing, *signerEmail)
		if signerId == nil {
			log.Println("Can't find signer id in keyring")
			return
		}

		err = signerId.PrivateKey.Decrypt([]byte(""))
		if err != nil {
			log.Println("Can't decrypt private key, " + err.Error())
			return
		}
	}

	updateChan := make(chan UpdateRequest)

	aptRepo := aptRepo{
		repoBase,
		poolRegexp,
	}

	blobStore := Sha1Store(*storeDir, *tmpDir, 3)

	aptGenerator := NewAptBlobArchiveGenerator(
		&aptRepo,
		privRing,
		signerId,
		blobStore,
	)

	uploadSessionManager := NewUploadSessionManager(
		expire,
		tmpDir,
		blobStore,
		NewScriptHook(uploadHook),
		*validateChanges,
		*validateChangesSufficient,
		*validateDebs,
		pubRing,
		updateChan,
	)

	server := &AptServer{
		MaxReqs:        *maxReqs,
		CookieName:     *cookieName,
		PreAftpHook:    NewScriptHook(preAftpHook),
		PostAftpHook:   NewScriptHook(postAftpHook),
		AcceptLoneDebs: *acceptLoneDebs,

		Repo:           &aptRepo,
		AptGenerator:   aptGenerator,
		SessionManager: uploadSessionManager,
		UpdateChannel:  updateChan,
	}

	server.InitAptServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)

	server.Register(mux)
	http.ListenAndServe(*listenAddress, mux)
}
