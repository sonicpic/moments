package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	projectdb "github.com/kingwrcy/moments/db"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestPublicProfileEndpointsDoNotExposeEmail(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&projectdb.User{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	user := projectdb.User{
		Id:        1,
		Username:  "admin",
		Nickname:  "管理员",
		Password:  "not-returned",
		Email:     "private@example.com",
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	handler := UserHandler{base: BaseHandler{db: database}}
	e := echo.New()

	for _, test := range []struct {
		name string
		call func(*echo.Echo, *UserHandler) *httptest.ResponseRecorder
	}{
		{
			name: "default public profile",
			call: func(e *echo.Echo, h *UserHandler) *httptest.ResponseRecorder {
				recorder := httptest.NewRecorder()
				context := CustomContext{Context: e.NewContext(httptest.NewRequest("POST", "/api/user/profile", nil), recorder)}
				if err := h.Profile(context); err != nil {
					t.Fatal(err)
				}
				return recorder
			},
		},
		{
			name: "named public profile",
			call: func(e *echo.Echo, h *UserHandler) *httptest.ResponseRecorder {
				recorder := httptest.NewRecorder()
				context := e.NewContext(httptest.NewRequest("POST", "/api/user/profile/admin", nil), recorder)
				context.SetPath("/api/user/profile/:username")
				context.SetParamNames("username")
				context.SetParamValues("admin")
				if err := h.ProfileForUser(context); err != nil {
					t.Fatal(err)
				}
				return recorder
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := test.call(e, &handler)
			var response struct {
				Code int            `json:"code"`
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != 0 {
				t.Fatalf("unexpected response: %s", recorder.Body.String())
			}
			if _, found := response.Data["email"]; found {
				t.Fatalf("public profile exposed email: %s", recorder.Body.String())
			}
			if _, found := response.Data["password"]; found {
				t.Fatalf("public profile exposed password: %s", recorder.Body.String())
			}
		})
	}
}
