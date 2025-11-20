package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGetScript(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/scripts/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		resp := V1Script{
			Id:    1,
			Title: "diagnostic script",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	httpClient := server.Client()

	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer test-token")
		return nil
	}

	baseURL := server.URL

	t.Run("raw response", func(t *testing.T) {
		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		resp, err := client.GetScript(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetScript failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		var payload V1Script
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if payload.Id != 1 || payload.Title != "diagnostic script" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	})

	t.Run("typed response", func(t *testing.T) {
		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.GetScriptWithResponse(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetScriptWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode())
		}

		if resp.JSON200 == nil {
			t.Fatal("expected JSON200 payload, got nil")
		}

		v1Script, err := resp.JSON200.AsV1Script()
		if err != nil {
			t.Fatalf("err converting to v1 script")
		}

		if v1Script.Id != 1 || v1Script.Title != "diagnostic script" {
			t.Fatalf("unexpected v1 script: %+v", v1Script)
		}
	})
}

func TestInvokeLegacyAction(t *testing.T) {
	handler := http.NewServeMux()
	legacyHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.URL.Query().Get("version") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("action") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		action := r.URL.Query().Get("action")
		switch action {
		case "CreateScript":
			w.Header().Set("Content-Type", "application/json")
			resp := V1Script{
				Id:    42,
				Title: r.URL.Query().Get("title"),
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		case "CreateScriptAttachment":
			w.Header().Set("Content-Type", "application/json")
			resp := "note.txt"
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		case "EditScript":
			w.Header().Set("Content-Type", "application/json")
			resp := V1Script{
				Id:    42,
				Title: r.URL.Query().Get("title"),
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		case "CopyScript":
			w.Header().Set("Content-Type", "application/json")
			resp := V1Script{
				Id:    99,
				Title: r.URL.Query().Get("destination_title"),
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("failed to encode response: %v", err)
			}
		case "RemoveScript":
			w.WriteHeader(http.StatusNoContent)
		case "RemoveScriptAttachment":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}
	handler.HandleFunc("/api", legacyHandler)
	handler.HandleFunc("/api/", legacyHandler)

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	httpClient := server.Client()

	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer test-token")
		return nil
	}

	baseURL := server.URL

	t.Run("missing version", func(t *testing.T) {
		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		params := &InvokeLegacyActionParams{
			Version: "",
			Action:  "CreateScript",
		}

		queryValues := url.Values{
			"title": []string{"example"},
			"code":  []string{"ZWNobyAiSGVsbG8i"},
		}

		resp, err := client.InvokeLegacyAction(
			context.Background(),
			params,
			EncodeQueryRequestEditor(queryValues),
		)
		if err != nil {
			t.Fatalf("InvokeLegacyAction returned error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 when version missing, got %d", resp.StatusCode)
		}
	})

	t.Run("missing action", func(t *testing.T) {
		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		params := &InvokeLegacyActionParams{
			Version: "2011-08-01",
			Action:  "",
		}

		queryValues := url.Values{
			"title": []string{"example"},
			"code":  []string{"ZWNobyAiSGVsbG8i"},
		}

		resp, err := client.InvokeLegacyAction(
			context.Background(),
			params,
			EncodeQueryRequestEditor(queryValues),
		)
		if err != nil {
			t.Fatalf("InvokeLegacyAction returned error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 when action missing, got %d", resp.StatusCode)
		}
	})

	t.Run("create script raw response", func(t *testing.T) {
		values := url.Values{
			"title": []string{"new script"},
			"code":  []string{"ZWNobyAiSGVsbG8i"},
		}

		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		resp, err := client.InvokeLegacyAction(context.Background(), LegacyActionParams("CreateScript"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyAction failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode)
		}

		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		if !strings.Contains(string(payload), "new script") {
			t.Fatalf("unexpected response payload: %s", payload)
		}
	})

	t.Run("create script typed response", func(t *testing.T) {
		values := url.Values{
			"title": []string{"new script"},
			"code":  []string{"ZWNobyAiSGVsbG8i"},
		}

		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.InvokeLegacyActionWithResponse(context.Background(), LegacyActionParams("CreateScript"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyActionWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode())
		}

		if resp.JSON200 == nil {
			t.Fatal("expected JSON200 payload, got nil")
		}

		script, err := resp.JSON200.AsScriptResult()
		if err != nil {
			t.Fatalf("failed to decode script from union: %v", err)
		}

		parsedScript, err := script.AsV1Script()
		if err != nil {
			t.Fatalf("failed to decode script from union: %v", err)
		}

		if parsedScript.Title != "new script" || parsedScript.Id != 42 {
			t.Fatalf("unexpected script payload: %+v", parsedScript)
		}
	})

	t.Run("edit script typed response", func(t *testing.T) {
		values := url.Values{
			"script_id": []string{"42"},
			"title":     []string{"edited title"},
		}

		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.InvokeLegacyActionWithResponse(context.Background(), LegacyActionParams("EditScript"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyActionWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode())
		}

		if resp.JSON200 == nil {
			t.Fatal("expected JSON200 payload, got nil")
		}

		script, err := resp.JSON200.AsScriptResult()
		if err != nil {
			t.Fatalf("failed to decode script from union: %v", err)
		}

		parsedScript, err := script.AsV1Script()
		if err != nil {
			t.Fatalf("failed to decode script from union: %v", err)
		}

		if parsedScript.Title != "edited title" {
			t.Fatalf("unexpected script payload: %+v", parsedScript)
		}
	})

	t.Run("copy script typed response", func(t *testing.T) {
		values := url.Values{
			"script_id":         []string{"42"},
			"destination_title": []string{"copy title"},
		}

		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.InvokeLegacyActionWithResponse(context.Background(), LegacyActionParams("CopyScript"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyActionWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode())
		}

		if resp.JSON200 == nil {
			t.Fatal("expected JSON200 payload, got nil")
		}

		script, err := resp.JSON200.AsScriptResult()
		if err != nil {
			t.Fatalf("failed to decode script from union: %v", err)
		}

		parsedScript, err := script.AsV1Script()
		if err != nil {
			t.Fatalf("failed to decode script from union: %v", err)
		}

		if parsedScript.Id != 99 || parsedScript.Title != "copy title" {
			t.Fatalf("unexpected script payload: %+v", parsedScript)
		}
	})

	t.Run("remove script raw response", func(t *testing.T) {
		values := url.Values{
			"script_id": []string{"42"},
		}

		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		resp, err := client.InvokeLegacyAction(context.Background(), LegacyActionParams("RemoveScript"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyAction failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected HTTP 204 but received %d", resp.StatusCode)
		}
	})

	t.Run("remove attachment typed response", func(t *testing.T) {
		values := url.Values{
			"script_id": []string{"42"},
			"filename":  []string{"note.txt"},
		}

		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.InvokeLegacyActionWithResponse(context.Background(), LegacyActionParams("RemoveScriptAttachment"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyActionWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusNoContent {
			t.Fatalf("expected HTTP 204 but received %d", resp.StatusCode())
		}

		if resp.JSON200 != nil {
			t.Fatalf("expected no payload for 204 response, got %#v", resp.JSON200)
		}
	})

	t.Run("create attachment typed response", func(t *testing.T) {
		values := url.Values{
			"script_id": []string{"42"},
			"file":      []string{"note.txt$$Zm9v"},
		}

		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.InvokeLegacyActionWithResponse(context.Background(), LegacyActionParams("CreateScriptAttachment"), EncodeQueryRequestEditor(values))
		if err != nil {
			t.Fatalf("InvokeLegacyActionWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode())
		}

		if resp.JSON200 == nil {
			t.Fatal("expected JSON200 payload, got nil")
		}

		attachment, err := resp.JSON200.AsLegacyScriptAttachment()
		if err != nil {
			t.Fatalf("failed to decode attachment result: %v", err)
		}

		if attachment != "note.txt" {
			t.Fatalf("unexpected attachment result: %+v", attachment)
		}
	})

}

func TestArchiveAndRedactScript(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/scripts/1:archive", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler.HandleFunc("/api/scripts/1:redact", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	httpClient := server.Client()

	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer test-token")
		return nil
	}

	baseURL := server.URL

	t.Run("archive raw", func(t *testing.T) {
		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		resp, err := client.ArchiveScript(context.Background(), 1)
		if err != nil {
			t.Fatalf("ArchiveScript failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected HTTP 204 but received %d", resp.StatusCode)
		}
	})

	t.Run("archive typed", func(t *testing.T) {
		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.ArchiveScriptWithResponse(context.Background(), 1)
		if err != nil {
			t.Fatalf("ArchiveScriptWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusNoContent {
			t.Fatalf("expected HTTP 204 but received %d", resp.StatusCode())
		}

		if len(resp.Body) > 0 {
			t.Fatalf("expected empty body for 204 response, got %q", resp.Body)
		}
	})

	t.Run("redact raw", func(t *testing.T) {
		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		resp, err := client.RedactScript(context.Background(), 1)
		if err != nil {
			t.Fatalf("RedactScript failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected HTTP 204 but received %d", resp.StatusCode)
		}
	})

	t.Run("redact typed", func(t *testing.T) {
		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.RedactScriptWithResponse(context.Background(), 1)
		if err != nil {
			t.Fatalf("RedactScriptWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusNoContent {
			t.Fatalf("expected HTTP 204 but received %d", resp.StatusCode())
		}

		if len(resp.Body) > 0 {
			t.Fatalf("expected empty body for 204 response, got %q", resp.Body)
		}
	})
}

func TestGetScriptAttachment(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/scripts/1/attachments/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode("file contents"); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})
	handler.HandleFunc("/api/scripts/99/attachments/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(map[string]any{"message": "not found"}); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	server := httptest.NewTLSServer(handler)
	defer server.Close()

	httpClient := server.Client()

	authEditor := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer test-token")
		return nil
	}

	baseURL := server.URL

	t.Run("raw response", func(t *testing.T) {
		client, err := NewClient(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client: %v", err)
		}

		resp, err := client.GetScriptAttachment(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("GetScriptAttachment failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}

		var attachment string
		if err := json.Unmarshal(body, &attachment); err != nil {
			t.Fatalf("failed to decode attachment: %v", err)
		}

		if attachment != "file contents" {
			t.Fatalf("unexpected attachment: %s", attachment)
		}
	})

	t.Run("typed response", func(t *testing.T) {
		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.GetScriptAttachmentWithResponse(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("GetScriptAttachmentWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected HTTP 200 but received %d", resp.StatusCode())
		}

		if resp.JSON200 == nil {
			t.Fatal("expected JSON200 payload, got nil")
		}

		if *resp.JSON200 != "file contents" {
			t.Fatalf("unexpected attachment payload: %s", *resp.JSON200)
		}
	})

	t.Run("not found typed response", func(t *testing.T) {
		client, err := NewClientWithResponses(baseURL, WithHTTPClient(httpClient), WithRequestEditorFn(authEditor))
		if err != nil {
			t.Fatalf("failed to init client with responses: %v", err)
		}

		resp, err := client.GetScriptAttachmentWithResponse(context.Background(), 99, 1)
		if err != nil {
			t.Fatalf("GetScriptAttachmentWithResponse failed: %v", err)
		}

		if resp.StatusCode() != http.StatusNotFound {
			t.Fatalf("expected HTTP 404 but received %d", resp.StatusCode())
		}

		if resp.JSON404 == nil || resp.JSON404.Message == nil || *resp.JSON404.Message != "not found" {
			t.Fatalf("unexpected 404 payload: %#v", resp.JSON404)
		}
	})
}
