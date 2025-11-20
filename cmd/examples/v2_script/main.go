// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/jansdhillon/landscape-go-api-client/client"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseURL := os.Getenv("LANDSCAPE_BASE_URL")
	if baseURL == "" {
		log.Fatalf("base URL not set")
	}
	ak := os.Getenv("LANDSCAPE_ACCESS_KEY")
	if ak == "" {
		log.Fatalf("access key not set")
	}

	sk := os.Getenv("LANDSCAPE_SECRET_KEY")
	if sk == "" {
		log.Fatalf("secret key not set")
	}

	landscapeAPIClient, err := client.NewLandscapeAPIClient(
		baseURL,
		client.NewAccessKeyProvider(ak, sk),
	)

	if err != nil {
		log.Fatalf("failed to create Landscape API client: %v", err)
	}

	// Create a V2 script
	createParams := client.LegacyActionParams("CreateScript")
	rawCode := "#!/bin/bash\n \"hello\" > /home/ubuntu/hello.txt"
	enc := base64.StdEncoding.EncodeToString([]byte(rawCode))
	queryArgsEditorFn := client.EncodeQueryRequestEditor(url.Values{
		"title":       []string{rand.Text()},
		"code":        []string{enc},
		"script_type": []string{"V2"},
	})
	createdScriptRes, err := landscapeAPIClient.InvokeLegacyActionWithResponse(ctx, createParams, queryArgsEditorFn)
	if err != nil {
		log.Fatalf("failed to invoke legacy action: %v", err)
	}

	log.Printf("raw create script response: %s", createdScriptRes.Body)
	if createdScriptRes.JSON200 == nil {
		if createdScriptRes.JSON404 != nil {
			log.Fatalf("error getting script: %s", createdScriptRes.Status())
		}
	}

	script, err := createdScriptRes.JSON200.AsScriptResult()
	if err != nil {
		log.Fatalf("failed to parse response as script: %v", err)
	}

	createdScript, err := script.AsV1Script()
	if err != nil {
		log.Fatalf("failed to parse response as V2 script: %v", err)
	}

	editParams := client.LegacyActionParams("EditScript")
	raw := "#!/bin/bash\necho \"newcode\" > /home/ubuntu/myscript.txt"
	enc = base64.StdEncoding.EncodeToString([]byte(raw))
	queryArgsEditorFn = client.EncodeQueryRequestEditor(url.Values{
		"script_id": []string{strconv.Itoa(createdScript.Id)},
		"username":  []string{"jim"},
		"code":      []string{enc},
	})

	res, err := landscapeAPIClient.InvokeLegacyActionWithResponse(ctx, editParams, queryArgsEditorFn)
	if err != nil {
		log.Fatalf("failed to invoke legacy action: %v", err)
	}

	log.Printf("raw edit script response: %s", res.Body)
	if res.JSON200 == nil {
		log.Fatalf("failed to edit script: %s", res.Status())
	}

	script, err = res.JSON200.AsScriptResult()
	if err != nil {
		log.Fatalf("failed to parse response as script: %s", err)
	}

	editedScript, err := script.AsV2Script()
	if err != nil {
		log.Fatalf("failed to script as V2 script: %s", err)
	}

	log.Printf("edited script title: %s", editedScript.Title)
	if editedScript.Attachments != nil {
		log.Printf("edited script attachments: %+v", *editedScript.Attachments)
	}
	if editedScript.Attachments != nil {
		log.Printf("edited script attachments count: %d", len(*editedScript.Attachments))
		for i, attachment := range *editedScript.Attachments {
			log.Printf("attachment %d: %+v", i, attachment)
		}
	}
	if editedScript.CreatedBy != nil {
		log.Printf("edited created by id: %d", *editedScript.CreatedBy.Id)
		log.Printf("edited created by name: %s", *editedScript.CreatedBy.Name)
	}

}
