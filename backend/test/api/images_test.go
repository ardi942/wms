package api_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"wms/backend/internal/testkit"
)

func pngImage(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func jpgImage(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	img.Set(0, 0, color.RGBA{G: 255, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func uploadImage(t *testing.T, env *testkit.Env, path, token, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	env.Router.ServeHTTP(rec, req)
	return rec
}

func TestProductImageLifecycle(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	supervisor := loginToken(t, env, "supervisor@example.com", testkit.Password)
	path := "/api/v1/products/" + env.ProductID.String() + "/image"
	first := pngImage(t)

	rec := uploadImage(t, env, path, admin, "product.png", first)
	requireStatus(t, rec, http.StatusOK)
	var uploaded struct {
		Data struct {
			ImageURL       *string `json:"image_url"`
			ImageUpdatedAt *string `json:"image_updated_at"`
		} `json:"data"`
	}
	mustDecode(t, rec, &uploaded)
	if uploaded.Data.ImageURL == nil || *uploaded.Data.ImageURL != "/products/"+env.ProductID.String()+"/image" {
		t.Fatalf("unexpected image_url: %#v", uploaded.Data.ImageURL)
	}
	if uploaded.Data.ImageUpdatedAt == nil {
		t.Fatal("missing image_updated_at")
	}

	for name, token := range map[string]string{"department": dept, "supervisor": supervisor, "admin": admin} {
		t.Run("get "+name, func(t *testing.T) {
			rec := env.Request(http.MethodGet, path, nil, token, nil)
			requireStatus(t, rec, http.StatusOK)
			if got := rec.Header().Get("Content-Type"); got != "image/png" {
				t.Fatalf("content type = %q", got)
			}
			if got := rec.Header().Get("Cache-Control"); got != "private, max-age=3600" {
				t.Fatalf("cache control = %q", got)
			}
			if !bytes.Equal(rec.Body.Bytes(), first) {
				t.Fatal("served image differs from upload")
			}
		})
	}

	rec = env.Request(http.MethodGet, path, nil, admin, map[string]string{"Range": "bytes=0-1"})
	requireStatus(t, rec, http.StatusPartialContent)
	if !bytes.Equal(rec.Body.Bytes(), first[:2]) {
		t.Fatalf("range body = %v, want %v", rec.Body.Bytes(), first[:2])
	}

	second := jpgImage(t)
	rec = uploadImage(t, env, path, admin, "product.jpg", second)
	requireStatus(t, rec, http.StatusOK)
	rec = env.Request(http.MethodGet, path, nil, dept, nil)
	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "image/jpeg" {
		t.Fatalf("content type after replace = %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), second) {
		t.Fatal("served image differs from replacement")
	}
	assertStoredFiles(t, env, "products", 1)

	var auditJSON string
	if err := env.DB.Raw(`SELECT changed_fields::text FROM audit_logs WHERE entity_id = ? AND action = 'PRODUCT_UPDATED' ORDER BY created_at DESC LIMIT 1`, env.ProductID).Scan(&auditJSON).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(auditJSON, "image") {
		t.Fatalf("audit changed_fields missing image: %s", auditJSON)
	}
	before, after := auditImageValues(t, env, env.ProductID, "PRODUCT_UPDATED")
	if before == nil || after == nil || *before == *after {
		t.Fatalf("expected distinct audit image timestamps, before=%v after=%v", before, after)
	}

	rec = env.Request(http.MethodPatch, "/api/v1/products/"+env.ProductID.String(), map[string]any{"is_active": false}, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	rec = env.Request(http.MethodGet, path, nil, dept, nil)
	requireStatus(t, rec, http.StatusNotFound)

	rec = env.Request(http.MethodDelete, path, nil, admin, nil)
	requireStatus(t, rec, http.StatusNoContent)
	rec = env.Request(http.MethodGet, path, nil, admin, nil)
	requireStatus(t, rec, http.StatusNotFound)
	before, after = auditImageValues(t, env, env.ProductID, "PRODUCT_UPDATED")
	if before == nil || after != nil {
		t.Fatalf("expected audit image removal, before=%v after=%v", before, after)
	}
	assertStoredFiles(t, env, "products", 0)
}

func TestUserImageLifecycle(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	deptID := env.Users["dept"].ID
	path := "/api/v1/users/" + deptID.String() + "/image"
	data := pngImage(t)

	rec := uploadImage(t, env, path, admin, "avatar.png", data)
	requireStatus(t, rec, http.StatusOK)
	var uploaded struct {
		Data struct {
			ImageURL *string `json:"image_url"`
		} `json:"data"`
	}
	mustDecode(t, rec, &uploaded)
	if uploaded.Data.ImageURL == nil || *uploaded.Data.ImageURL != "/users/"+deptID.String()+"/image" {
		t.Fatalf("unexpected image_url: %#v", uploaded.Data.ImageURL)
	}

	rec = env.Request(http.MethodGet, path, nil, dept, nil)
	requireStatus(t, rec, http.StatusOK)
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Fatal("served avatar differs from upload")
	}
	rec = env.Request(http.MethodGet, path, nil, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	otherPath := "/api/v1/users/" + env.Users["admin"].ID.String() + "/image"
	rec = env.Request(http.MethodGet, otherPath, nil, dept, nil)
	requireStatus(t, rec, http.StatusForbidden)

	var auditJSON string
	if err := env.DB.Raw(`SELECT changed_fields::text FROM audit_logs WHERE entity_id = ? AND action = 'USER_PROFILE_UPDATED' ORDER BY created_at DESC LIMIT 1`, deptID).Scan(&auditJSON).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(auditJSON, "image") {
		t.Fatalf("audit changed_fields missing image: %s", auditJSON)
	}

	rec = env.Request(http.MethodDelete, path, nil, admin, nil)
	requireStatus(t, rec, http.StatusNoContent)
	rec = env.Request(http.MethodGet, path, nil, dept, nil)
	requireStatus(t, rec, http.StatusNotFound)
	assertStoredFiles(t, env, "users", 0)
}

func TestImageValidationErrors(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	path := "/api/v1/products/" + env.ProductID.String() + "/image"

	rec := uploadImage(t, env, path, admin, "image.txt", []byte("not an image"))
	requireStatus(t, rec, http.StatusUnprocessableEntity)
	var invalid struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	mustDecode(t, rec, &invalid)
	if invalid.Error.Code != "VALIDATION" || invalid.Error.Details["image"] == nil {
		t.Fatalf("unexpected validation response: %s", rec.Body.String())
	}

	rec = uploadImage(t, env, path, admin, "large.png", bytes.Repeat([]byte{0}, (2<<20)+1))
	requireStatus(t, rec, http.StatusRequestEntityTooLarge)

	oversizedBody := bytes.Repeat([]byte{0}, (2<<20)+(600<<10))
	rec = uploadImage(t, env, path, admin, "body.png", oversizedBody)
	requireStatus(t, rec, http.StatusRequestEntityTooLarge)

	for name, data := range map[string][]byte{
		"svg":     []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`),
		"gif":     []byte("GIF89a"),
		"corrupt": append([]byte("\x89PNG\r\n\x1a\n"), []byte("not really a png")...),
	} {
		t.Run(name, func(t *testing.T) {
			rec := uploadImage(t, env, path, admin, name, data)
			requireStatus(t, rec, http.StatusUnprocessableEntity)
			requireErrorCode(t, rec, "VALIDATION")
		})
	}

	rec = env.Request(http.MethodPost, path, nil, admin, nil)
	requireStatus(t, rec, http.StatusUnprocessableEntity)
}

func TestImageMutationPermissions(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	supervisor := loginToken(t, env, "supervisor@example.com", testkit.Password)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	productPath := "/api/v1/products/" + env.ProductID.String() + "/image"
	userPath := "/api/v1/users/" + env.Users["dept"].ID.String() + "/image"
	image := pngImage(t)

	for name, token := range map[string]string{"supervisor": supervisor, "department": dept} {
		t.Run(name+" cannot mutate product image", func(t *testing.T) {
			rec := uploadImage(t, env, productPath, token, "product.png", image)
			requireStatus(t, rec, http.StatusForbidden)
			rec = env.Request(http.MethodDelete, productPath, nil, token, nil)
			requireStatus(t, rec, http.StatusForbidden)
		})
	}
	for name, token := range map[string]string{"supervisor": supervisor, "department": dept} {
		t.Run(name+" cannot mutate user image", func(t *testing.T) {
			rec := uploadImage(t, env, userPath, token, "avatar.png", image)
			requireStatus(t, rec, http.StatusForbidden)
			rec = env.Request(http.MethodDelete, userPath, nil, token, nil)
			requireStatus(t, rec, http.StatusForbidden)
		})
	}

	missing := uuid.NewString()
	for name, path := range map[string]string{
		"product": "/api/v1/products/" + missing + "/image",
		"user":    "/api/v1/users/" + missing + "/image",
	} {
		t.Run(name+" missing entity", func(t *testing.T) {
			rec := uploadImage(t, env, path, admin, "image.png", image)
			requireStatus(t, rec, http.StatusNotFound)
			rec = env.Request(http.MethodDelete, path, nil, admin, nil)
			requireStatus(t, rec, http.StatusNotFound)
		})
	}
}

func TestImageMetadataInAPIResponses(t *testing.T) {
	env := testkit.New(t)
	admin := loginToken(t, env, "admin@example.com", testkit.Password)
	dept := loginToken(t, env, "dept@example.com", testkit.Password)
	productPath := "/api/v1/products/" + env.ProductID.String() + "/image"
	userPath := "/api/v1/users/" + env.Users["dept"].ID.String() + "/image"

	rec := uploadImage(t, env, productPath, admin, "product.png", pngImage(t))
	requireStatus(t, rec, http.StatusOK)
	rec = uploadImage(t, env, userPath, admin, "avatar.png", pngImage(t))
	requireStatus(t, rec, http.StatusOK)

	rec = env.Request(http.MethodGet, "/api/v1/products/"+env.ProductID.String(), nil, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	var detail struct {
		Data struct {
			ImageURL       *string `json:"image_url"`
			ImageUpdatedAt *string `json:"image_updated_at"`
		} `json:"data"`
	}
	mustDecode(t, rec, &detail)
	if detail.Data.ImageURL == nil || *detail.Data.ImageURL != "/products/"+env.ProductID.String()+"/image" || detail.Data.ImageUpdatedAt == nil {
		t.Fatalf("product detail missing image metadata: %s", rec.Body.String())
	}

	rec = env.Request(http.MethodGet, "/api/v1/products", nil, admin, nil)
	requireStatus(t, rec, http.StatusOK)
	var listed struct {
		Data []struct {
			ID             string  `json:"id"`
			ImageURL       *string `json:"image_url"`
			ImageUpdatedAt *string `json:"image_updated_at"`
		} `json:"data"`
	}
	mustDecode(t, rec, &listed)
	found := false
	for _, item := range listed.Data {
		if item.ID != env.ProductID.String() {
			continue
		}
		found = true
		if item.ImageURL == nil || item.ImageUpdatedAt == nil {
			t.Fatalf("product list missing image metadata: %s", rec.Body.String())
		}
	}
	if !found {
		t.Fatal("uploaded product not present in list")
	}

	rec = env.Request(http.MethodGet, "/api/v1/auth/me", nil, dept, nil)
	requireStatus(t, rec, http.StatusOK)
	var me struct {
		Data struct {
			User struct {
				ImageURL       *string `json:"image_url"`
				ImageUpdatedAt *string `json:"image_updated_at"`
			} `json:"user"`
		} `json:"data"`
	}
	mustDecode(t, rec, &me)
	if me.Data.User.ImageURL == nil || *me.Data.User.ImageURL != "/users/"+env.Users["dept"].ID.String()+"/image" || me.Data.User.ImageUpdatedAt == nil {
		t.Fatalf("auth/me missing image metadata: %s", rec.Body.String())
	}
}

func auditImageValues(t *testing.T, env *testkit.Env, entityID uuid.UUID, action string) (before, after *string) {
	t.Helper()
	var rows []struct {
		Before *string
		After  *string
	}
	if err := env.DB.Raw(`
		SELECT before_data->>'image' AS before, after_data->>'image' AS after
		FROM audit_logs
		WHERE entity_id = ? AND action = ? AND 'image' = ANY(changed_fields)
		ORDER BY created_at DESC, id DESC
		LIMIT 1`, entityID, action).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("no image audit entry found")
	}
	return rows[0].Before, rows[0].After
}

func assertStoredFiles(t *testing.T, env *testkit.Env, prefix string, want int) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(env.Config.UploadDir, prefix))
	if err != nil {
		if os.IsNotExist(err) && want == 0 {
			return
		}
		t.Fatal(err)
	}
	if len(entries) != want {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		raw, _ := json.Marshal(names)
		t.Fatalf("stored files in %s = %s, want %d", prefix, raw, want)
	}
}
