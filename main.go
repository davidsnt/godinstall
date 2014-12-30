package main

// Package GoDInstall implements a web service for serving, and manipulating
// debian Apt repositories. The original motivation was to provide a synchronous
// interface for package upload. A package is available for download from the
// repository at the point when the server confirms the package has been
// uploaded.
//   It is primarily aimed at use in continuous delivery processes.

import (
	_ "expvar"
	"os"

	"code.google.com/p/go.crypto/openpgp"
	"github.com/codegangsta/cli"
)

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
	app := cli.NewApp()
	app.Name = "godinstall"
	app.Usage = "dynamic apt repository server"
	app.Version = godinstallVersion

	app.Commands = []cli.Command{
		cli.Command{
			Name: "serve",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "l, listen",
					Value: ":3000",
					Usage: "The listen address",
				},
				cli.StringFlag{
					Name:  "t, ttl",
					Value: "60s",
					Usage: "Upload session will be terminated after the TTL",
				},
				cli.IntFlag{
					Name:  "max-requests",
					Value: 4,
					Usage: "Maximum concurrent requests",
				},
				cli.StringFlag{
					Name:  "repo-base",
					Value: "",
					Usage: "Location of repository root",
				},
				cli.StringFlag{
					Name:  "cookie-name",
					Value: "godinstall-sess",
					Usage: "Name for the sessio cookie",
				},
				cli.StringFlag{
					Name:  "upload-hook",
					Value: "",
					Usage: "Script to run after for each uploaded file",
				},
				cli.StringFlag{
					Name:  "pre-gen-hook",
					Value: "",
					Usage: "Script to run before archive regeneration",
				},
				cli.StringFlag{
					Name:  "post-gen-hook",
					Value: "",
					Usage: "Script to run after archive regeneration",
				},
				cli.StringFlag{
					Name:  "pool-pattern",
					Value: "[a-z]|lib[a-z]",
					Usage: "A pattern to match package prefixes to split into directories in the pool",
				},
				cli.BoolTFlag{
					Name:  "verify-changes",
					Usage: "Verify signatures on changes files",
				},
				cli.BoolTFlag{
					Name:  "verify-changes-sufficient",
					Usage: "If we are given a signed chnages file, we wont verify individual debs",
				},
				cli.BoolTFlag{
					Name:  "accept-lone-debs",
					Usage: "Accept individual debs for upload",
				},
				cli.BoolTFlag{
					Name:  "verify-debs",
					Usage: "Verify signatures on deb files",
				},
				cli.StringFlag{
					Name:  "gpg-pubring",
					Value: "",
					Usage: "Public keyring file",
				},
				cli.StringFlag{
					Name:  "gpg-privring",
					Value: "",
					Usage: "Private keyring file",
				},
				cli.StringFlag{
					Name:  "signer-email",
					Value: "",
					Usage: "Key Email to use for signing releases",
				},
				cli.StringFlag{
					Name:  "prune",
					Value: ".*_*-*",
					Usage: "Rules for package pruning",
				},
				cli.BoolFlag{
					Name:  "auto-trim",
					Usage: "Automatically trim branch history",
				},
				cli.IntFlag{
					Name:  "auto-trim-length",
					Value: 10,
					Usage: "Rules for package pruning",
				},
			},
			Usage:  "run a repository server",
			Action: CmdServe,
		},
		cli.Command{
			Name: "upload",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "url",
					Value: "http://localhost:3000/dists/master/upload",
					Usage: "URL to upload to",
				},
			},
			Usage:  "publish a package to a repository",
			Action: CmdUpload,
		},
	}

	app.Run(os.Args)
}
