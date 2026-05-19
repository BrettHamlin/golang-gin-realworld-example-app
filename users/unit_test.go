package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gothinkster/golang-gin-realworld-example-app/common"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var image_url = "https://golang.org/doc/gopher/frontpage.png"
var test_db *gorm.DB

func newUserModel() UserModel {
	return UserModel{
		ID:           2,
		Username:     "asd123!@#ASD",
		Email:        "wzt@g.cn",
		Bio:          "heheda",
		Image:        &image_url,
		PasswordHash: "",
	}
}

func userModelMocker(n int) []UserModel {
	var offset int64
	test_db.Model(&UserModel{}).Count(&offset)
	var ret []UserModel
	for i := int(offset) + 1; i <= int(offset)+n; i++ {
		image := fmt.Sprintf("http://image/%v.jpg", i)
		userModel := UserModel{
			Username: fmt.Sprintf("user%v", i),
			Email:    fmt.Sprintf("user%v@linkedin.com", i),
			Bio:      fmt.Sprintf("bio%v", i),
			Image:    &image,
		}
		userModel.setPassword("password123")
		test_db.Create(&userModel)
		ret = append(ret, userModel)
	}
	return ret
}

func TestUserModel(t *testing.T) {
	asserts := assert.New(t)

	//Testing UserModel's password feature
	userModel := newUserModel()
	err := userModel.checkPassword("")
	asserts.Error(err, "empty password should return err")

	userModel = newUserModel()
	err = userModel.setPassword("")
	asserts.Error(err, "empty password can not be set null")

	userModel = newUserModel()
	err = userModel.setPassword("asd123!@#ASD")
	asserts.NoError(err, "password should be set successful")
	asserts.Len(userModel.PasswordHash, 60, "password hash length should be 60")

	err = userModel.checkPassword("sd123!@#ASD")
	asserts.Error(err, "password should be checked and not validated")

	err = userModel.checkPassword("asd123!@#ASD")
	asserts.NoError(err, "password should be checked and validated")

	//Testing the following relationship between users
	users := userModelMocker(3)
	a := users[0]
	b := users[1]
	c := users[2]
	asserts.Equal(0, len(a.GetFollowings()), "GetFollowings should be right before following")
	asserts.Equal(false, a.isFollowing(b), "isFollowing relationship should be right at init")
	a.following(b)
	asserts.Equal(1, len(a.GetFollowings()), "GetFollowings should be right after a following b")
	asserts.Equal(true, a.isFollowing(b), "isFollowing should be right after a following b")
	a.following(c)
	asserts.Equal(2, len(a.GetFollowings()), "GetFollowings be right after a following c")
	asserts.EqualValues(b, a.GetFollowings()[0], "GetFollowings should be right")
	asserts.EqualValues(c, a.GetFollowings()[1], "GetFollowings should be right")
	a.unFollowing(b)
	asserts.Equal(1, len(a.GetFollowings()), "GetFollowings should be right after a unFollowing b")
	asserts.EqualValues(c, a.GetFollowings()[0], "GetFollowings should be right after a unFollowing b")
	asserts.Equal(false, a.isFollowing(b), "isFollowing should be right after a unFollowing b")
}

// Reset test DB and create new one with mock data
func resetDBWithMock() {
	common.TestDBFree(test_db)
	test_db = common.TestDBInit()
	AutoMigrate()
	userModelMocker(3)
}

// You could write the init logic like reset database code here
var unauthRequestTests = []struct {
	init           func(*http.Request)
	url            string
	method         string
	bodyData       string
	expectedCode   int
	responseRegexg string
	msg            string
}{
	//Testing will run one by one, so you can combine it to a user story till another init().
	//And you can modified the header or body in the func(req *http.Request) {}

	//---------------------   Testing for user register   ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
		},
		"/users/",
		"POST",
		`{"user":{"username": "wangzitian0","email": "wzt@gg.cn","password": "jakejxke"}}`,
		http.StatusCreated,
		`{"user":{"username":"wangzitian0","email":"wzt@gg.cn","bio":"","image":"","token":"([a-zA-Z0-9-_.]{115})"}}`,
		"valid data and should return StatusCreated",
	},
	{
		func(req *http.Request) {},
		"/users/",
		"POST",
		`{"user":{"username": "wangzitian0","email": "wzt@gg.cn","password": "jakejxke"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"database":"UNIQUE constraint failed: user_models.email"}}`,
		"duplicated data and should return StatusUnprocessableEntity",
	},
	{
		func(req *http.Request) {},
		"/users/",
		"POST",
		`{"user":{"username": "u","email": "wzt@gg.cn","password": "jakejxke"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Username":"{min: 4}"}}`,
		"too short username should return error",
	},
	{
		func(req *http.Request) {},
		"/users/",
		"POST",
		`{"user":{"username": "wangzitian0","email": "wzt@gg.cn","password": "j"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Password":"{min: 8}"}}`,
		"too short password should return error",
	},
	{
		func(req *http.Request) {},
		"/users/",
		"POST",
		`{"user":{"username": "wangzitian0","email": "wztgg.cn","password": "jakejxke"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Email":"{key: email}"}}`,
		"email invalid should return error",
	},

	//---------------------   Testing for user login   ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
		},
		"/users/login",
		"POST",
		`{"user":{"email": "user1@linkedin.com","password": "password123"}}`,
		http.StatusOK,
		`{"user":{"username":"user1","email":"user1@linkedin.com","bio":"bio1","image":"http://image/1.jpg","token":"([a-zA-Z0-9-_.]{115})"}}`,
		"right info login should return user",
	},
	{
		func(req *http.Request) {},
		"/users/login",
		"POST",
		`{"user":{"email": "user112312312@linkedin.com","password": "password123"}}`,
		http.StatusUnauthorized,
		`{"errors":{"login":"Not Registered email or invalid password"}}`,
		"email not exist should return error info",
	},
	{
		func(req *http.Request) {},
		"/users/login",
		"POST",
		`{"user":{"email": "user1@linkedin.com","password": "password126"}}`,
		http.StatusUnauthorized,
		`{"errors":{"login":"Not Registered email or invalid password"}}`,
		"password error should return error info",
	},
	{
		func(req *http.Request) {},
		"/users/login",
		"POST",
		`{"user":{"email": "user1@linkedin.com","password": "passw"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Password":"{min: 8}"}}`,
		"password too short should return error info",
	},
	{
		func(req *http.Request) {},
		"/users/login",
		"POST",
		`{"user":{"email": "user1@linkedin.com","password": "passw"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Password":"{min: 8}"}}`,
		"password too short should return error info",
	},

	//---------------------   Testing for self info get & auth module  ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
		},
		"/user/",
		"GET",
		``,
		http.StatusUnauthorized,
		``,
		"request should return 401 without token",
	},
	{
		func(req *http.Request) {
			req.Header.Set("Authorization", fmt.Sprintf("Tokee %v", common.GenToken(1)))
		},
		"/user/",
		"GET",
		``,
		http.StatusUnauthorized,
		``,
		"wrong token should return 401",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 1)
		},
		"/user/",
		"GET",
		``,
		http.StatusOK,
		`{"user":{"username":"user1","email":"user1@linkedin.com","bio":"bio1","image":"http://image/1.jpg","token":"([a-zA-Z0-9-_.]{115})"}}`,
		"request should return current user with token",
	},

	//---------------------   Testing for users' profile get   ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
		},
		"/profiles/user1",
		"GET",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`,
		"anonymous request should return profile with following=false",
	},
	{
		func(req *http.Request) {
			resetDBWithMock()
			common.HeaderTokenMock(req, 1)
		},
		"/profiles/user1",
		"GET",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`,
		"request should return self profile",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1",
		"GET",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`,
		"request should return correct other's profile",
	},

	//---------------------   Testing for users' profile update   ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
			common.HeaderTokenMock(req, 1)
		},
		"/profiles/user123",
		"GET",
		``,
		http.StatusNotFound,
		``,
		"user should not exist profile before changed",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 1)
		},
		"/user/",
		"PUT",
		`{"user":{"username":"user123","password": "password126","email":"user123@linkedin.com","bio":"bio123","image":"http://hehe/123.jpg"}}`,
		http.StatusOK,
		`{"user":{"username":"user123","email":"user123@linkedin.com","bio":"bio123","image":"http://hehe/123.jpg","token":"([a-zA-Z0-9-_.]{115})"}}`,
		"current user profile should be changed",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 1)
		},
		"/profiles/user123",
		"GET",
		``,
		http.StatusOK,
		`{"profile":{"username":"user123","bio":"bio123","image":"http://hehe/123.jpg","following":false}}`,
		"request should return self profile after changed",
	},
	{
		func(req *http.Request) {},
		"/users/login",
		"POST",
		`{"user":{"email": "user123@linkedin.com","password": "password126"}}`,
		http.StatusOK,
		`{"user":{"username":"user123","email":"user123@linkedin.com","bio":"bio123","image":"http://hehe/123.jpg","token":"([a-zA-Z0-9-_.]{115})"}}`,
		"user should login using new password after changed",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/user/",
		"PUT",
		`{"user":{"password": "pas"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Password":"{min: 8}"}}`,
		"current user profile should not be changed with error user info",
	},

	//---------------------   Testing for db errors   ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
			common.HeaderTokenMock(req, 4)
		},
		"/user/",
		"PUT",
		`{"password": "password321"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"Email":"{key: required}","Username":"{key: required}"}}`,
		"test database pk error for user update",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 0)
		},
		"/user/",
		"PUT",
		`{"user":{"username": "wangzitian0","email": "wzt@gg.cn","password": "jakejxke"}}`,
		http.StatusUnprocessableEntity,
		`{"errors":{"database":"WHERE conditions required"}}`,
		"cheat validator and test database connecting error for user update",
	},
	{
		func(req *http.Request) {
			common.TestDBFree(test_db)
			test_db = common.TestDBInit()

			test_db.AutoMigrate(&UserModel{})
			userModelMocker(3)
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1/follow",
		"POST",
		``,
		http.StatusUnprocessableEntity,
		`{"errors":{"database":"no such table: follow_models"}}`,
		"test database error for following",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1/follow",
		"DELETE",
		``,
		http.StatusUnprocessableEntity,
		`{"errors":{"database":"no such table: follow_models"}}`,
		"test database error for canceling following",
	},
	{
		func(req *http.Request) {
			resetDBWithMock()
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user666/follow",
		"POST",
		``,
		http.StatusNotFound,
		`{"errors":{"profile":"Invalid username"}}`,
		"following wrong user name should return errors",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user666/follow",
		"DELETE",
		``,
		http.StatusNotFound,
		`{"errors":{"profile":"Invalid username"}}`,
		"cancel following wrong user name should return errors",
	},

	//---------------------   Testing for user following   ---------------------
	{
		func(req *http.Request) {
			resetDBWithMock()
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1/follow",
		"POST",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":true}}`,
		"user follow another should work",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1",
		"GET",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":true}}`,
		"user follow another should make sure database changed",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1/follow",
		"DELETE",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`,
		"user cancel follow another should work",
	},
	{
		func(req *http.Request) {
			common.HeaderTokenMock(req, 2)
		},
		"/profiles/user1",
		"GET",
		``,
		http.StatusOK,
		`{"profile":{"username":"user1","bio":"bio1","image":"http://image/1.jpg","following":false}}`,
		"user cancel follow another should make sure database changed",
	},
}

func TestWithoutAuth(t *testing.T) {
	asserts := assert.New(t)
	//You could write the reset database code here if you want to create a database for this block
	//resetDB()

	r := gin.New()
	UsersRegister(r.Group("/users"))
	r.Use(AuthMiddleware(false))
	ProfileRetrieveRegister(r.Group("/profiles"))
	r.Use(AuthMiddleware(true))
	UserRegister(r.Group("/user"))
	ProfileRegister(r.Group("/profiles"))
	for _, testData := range unauthRequestTests {
		bodyData := testData.bodyData
		req, err := http.NewRequest(testData.method, testData.url, bytes.NewBufferString(bodyData))
		req.Header.Set("Content-Type", "application/json")
		asserts.NoError(err)

		testData.init(req)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		asserts.Equal(testData.expectedCode, w.Code, "Response Status - "+testData.msg)
		asserts.Regexp(testData.responseRegexg, w.Body.String(), "Response Content - "+testData.msg)
	}
}

func TestExtractTokenFromQueryParameter(t *testing.T) {
	asserts := assert.New(t)

	r := gin.New()
	r.Use(AuthMiddleware(false))
	r.GET("/test", func(c *gin.Context) {
		userID := c.MustGet("my_user_id").(uint)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	resetDBWithMock()

	// Test with access_token query parameter
	token := common.GenToken(1)
	req, _ := http.NewRequest("GET", "/test?access_token="+token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	asserts.Equal(http.StatusOK, w.Code, "Request with query token should succeed")
	asserts.Contains(w.Body.String(), `"user_id":1`, "User ID should be 1")
}

func TestAuthMiddlewareInvalidToken(t *testing.T) {
	asserts := assert.New(t)

	r := gin.New()
	r.Use(AuthMiddleware(true))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Test with invalid JWT token
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Token invalid.jwt.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	asserts.Equal(http.StatusUnauthorized, w.Code, "Invalid token should return 401")
}

func TestAuthMiddlewareNoToken(t *testing.T) {
	asserts := assert.New(t)

	r := gin.New()
	r.Use(AuthMiddleware(false))
	r.GET("/test", func(c *gin.Context) {
		userID := c.MustGet("my_user_id").(uint)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Test with no token (auto401=false should still proceed)
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	asserts.Equal(http.StatusOK, w.Code, "No token with auto401=false should proceed")
	asserts.Contains(w.Body.String(), `"user_id":0`, "User ID should be 0")
}

func newAPIUserFollowingRouter() *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false
	v1 := r.Group("/api")
	v1.Use(AuthMiddleware(false))
	ProfileRetrieveRegister(v1.Group("/profiles"))
	v1.Use(AuthMiddleware(true))
	UserRegister(v1.Group("/user"))
	ProfileRegister(v1.Group("/profiles"))
	return r
}

func performUserFollowingRequest(r *gin.Engine, path string, init func(*http.Request)) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", path, nil)
	if init != nil {
		init(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeProfilesRaw(t *testing.T, body string) []json.RawMessage {
	t.Helper()
	var response struct {
		Profiles json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("response should be valid JSON: %v; body=%q", err, body)
	}
	if string(response.Profiles) == "" {
		t.Fatalf("profiles key should be present in response body %q", body)
	}
	var profiles []json.RawMessage
	if err := json.Unmarshal(response.Profiles, &profiles); err != nil {
		t.Fatalf("profiles should be a JSON array: %v; raw=%s", err, response.Profiles)
	}
	return profiles
}

func decodeJSONBody(t *testing.T, body string) interface{} {
	t.Helper()
	var decoded interface{}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("response should be valid JSON: %v; body=%q", err, body)
	}
	return decoded
}

//harness:criterion=c-following-unauth-returns-401,c-following-unauth-trailing-slash-returns-401,c-following-route-registered-under-auth-middleware
func TestUserFollowingUnauthenticatedRequestsReturn401EmptyBody(t *testing.T) {
	asserts := assert.New(t)
	resetDBWithMock()
	r := newAPIUserFollowingRouter()

	for _, path := range []string{"/api/user/following", "/api/user/following/"} {
		w := performUserFollowingRequest(r, path, nil)
		asserts.Equal(http.StatusUnauthorized, w.Code, "unauthenticated request should be rejected for "+path)
		asserts.Equal("", w.Body.String(), "unauthenticated response body should be empty for "+path)
	}
}

//harness:criterion=c-following-invalid-token-returns-401
func TestUserFollowingInvalidTokenReturns401EmptyBody(t *testing.T) {
	asserts := assert.New(t)
	resetDBWithMock()
	r := newAPIUserFollowingRouter()

	w := performUserFollowingRequest(r, "/api/user/following", func(req *http.Request) {
		req.Header.Set("Authorization", "Token invalidtoken123")
	})

	asserts.Equal(http.StatusUnauthorized, w.Code, "malformed token should be rejected")
	asserts.Equal("", w.Body.String(), "malformed token response body should be empty")
}

//harness:criterion=c-following-empty-returns-200,c-following-empty-profiles-is-array-not-null,c-following-handler-reads-user-from-context
func TestUserFollowingEmptyReturnsProfilesArray(t *testing.T) {
	asserts := assert.New(t)
	resetDBWithMock()
	r := newAPIUserFollowingRouter()

	w := performUserFollowingRequest(r, "/api/user/following", func(req *http.Request) {
		common.HeaderTokenMock(req, 1)
	})

	asserts.Equal(http.StatusOK, w.Code, "authenticated user with no follows should succeed")
	var response struct {
		Profiles json.RawMessage `json:"profiles"`
	}
	asserts.NoError(json.Unmarshal(w.Body.Bytes(), &response), "response should be valid JSON")
	asserts.Equal(json.RawMessage("[]"), response.Profiles, "profiles should encode as an empty array")
	asserts.JSONEq(`{"profiles":[]}`, w.Body.String(), "empty following response should match contract body")
}

//harness:criterion=c-following-nonempty-returns-200,c-following-profile-following-field-true,c-following-profile-shape-matches-profileresponse,c-following-handler-reads-user-from-context,c-following-calls-getfollowings-model-method
func TestUserFollowingNonEmptyReturnsFollowedProfileResponses(t *testing.T) {
	asserts := assert.New(t)
	resetDBWithMock()
	r := newAPIUserFollowingRouter()
	follower, err := FindOneUser(&UserModel{Username: "user1"})
	asserts.NoError(err)
	followed, err := FindOneUser(&UserModel{Username: "user2"})
	asserts.NoError(err)
	asserts.NoError(follower.following(followed))

	w := performUserFollowingRequest(r, "/api/user/following", func(req *http.Request) {
		common.HeaderTokenMock(req, follower.ID)
	})

	asserts.Equal(http.StatusOK, w.Code, "authenticated user with follows should succeed")
	profiles := decodeProfilesRaw(t, w.Body.String())
	asserts.Len(profiles, 1, "profiles should contain exactly the followed users")

	var profile map[string]interface{}
	asserts.NoError(json.Unmarshal(profiles[0], &profile), "profile should decode as an object")
	asserts.Equal(followed.Username, profile["username"], "profile username should match followed user")
	asserts.Equal(followed.Bio, profile["bio"], "profile bio should match followed user")
	asserts.Equal(*followed.Image, profile["image"], "profile image should match followed user")
	asserts.Equal(true, profile["following"], "profile following field should be true")
	asserts.Equal(map[string]bool{
		"username":  true,
		"bio":       true,
		"image":     true,
		"following": true,
	}, mapKeys(profile), "profile should expose exactly the ProfileResponse fields")
}

//harness:criterion=c-following-trailing-slash-same-response
func TestUserFollowingTrailingSlashMatchesCanonicalResponse(t *testing.T) {
	asserts := assert.New(t)
	for _, testData := range []struct {
		name string
		init func(*testing.T) uint
	}{
		{
			name: "empty following list",
			init: func(t *testing.T) uint {
				return 1
			},
		},
		{
			name: "non-empty following list",
			init: func(t *testing.T) uint {
				t.Helper()
				follower, err := FindOneUser(&UserModel{Username: "user1"})
				asserts.NoError(err)
				followed, err := FindOneUser(&UserModel{Username: "user2"})
				asserts.NoError(err)
				asserts.NoError(follower.following(followed))
				return follower.ID
			},
		},
	} {
		t.Run(testData.name, func(t *testing.T) {
			resetDBWithMock()
			r := newAPIUserFollowingRouter()
			userID := testData.init(t)
			setAuth := func(req *http.Request) {
				common.HeaderTokenMock(req, userID)
			}

			withoutSlash := performUserFollowingRequest(r, "/api/user/following", setAuth)
			withSlash := performUserFollowingRequest(r, "/api/user/following/", setAuth)

			asserts.Equal(withoutSlash.Code, withSlash.Code, "status should match")
			asserts.True(reflect.DeepEqual(
				decodeJSONBody(t, withoutSlash.Body.String()),
				decodeJSONBody(t, withSlash.Body.String()),
			), "JSON body should match")
		})
	}
}

func mapKeys(values map[string]interface{}) map[string]bool {
	keys := make(map[string]bool, len(values))
	for key := range values {
		keys[key] = true
	}
	return keys
}

// This is a hack way to add test database for each case, as whole test will just share one database.
// You can read TestWithoutAuth's comment to know how to not share database each case.
func TestMain(m *testing.M) {
	test_db = common.TestDBInit()
	AutoMigrate()
	exitVal := m.Run()
	common.TestDBFree(test_db)
	os.Exit(exitVal)
}
