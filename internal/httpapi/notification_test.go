package httpapi

import (
"bytes"
"context"
"encoding/json"
"io"
"log/slog"
"net/http"
"net/http/httptest"
"testing"
"time"

"github.com/yibaiba/dramora/internal/domain"
"github.com/yibaiba/dramora/internal/repo"
"github.com/yibaiba/dramora/internal/service"
)

type testNotificationSetup struct {
authService         *service.AuthService
identityRepo        *repo.MemoryIdentityRepository
notificationRepo    *repo.MemoryNotificationRepository
notificationService *service.NotificationService
router              http.Handler
}

func setupNotificationTest(t *testing.T) *testNotificationSetup {
t.Helper()
logger := slog.New(slog.NewTextHandler(io.Discard, nil))
identityRepo := repo.NewMemoryIdentityRepository()
authService := service.NewAuthService(identityRepo, "test-secret", nil)
notificationRepo := repo.NewMemoryNotificationRepository()
notificationService := service.NewNotificationService(notificationRepo)
router := NewRouter(RouterConfig{
Logger:              logger,
Version:             "test",
AuthService:         authService,
NotificationService: notificationService,
})
return &testNotificationSetup{
authService:         authService,
identityRepo:        identityRepo,
notificationRepo:    notificationRepo,
notificationService: notificationService,
router:              router,
}
}

func issueNotificationSession(t *testing.T, setup *testNotificationSetup, userID, orgID, email, role string) string {
t.Helper()
identity, err := setup.identityRepo.CreateUserWithMembership(context.Background(), repo.CreateUserWithMembershipParams{
UserID:         userID,
OrganizationID: orgID,
Email:          email,
DisplayName:    "Notification Test " + role,
PasswordHash:   "$2a$10$abcdefghijklmnopqrstuv",
Role:           role,
})
if err != nil {
t.Fatalf("seed identity %s: %v", role, err)
}
session, err := setup.authService.IssueSessionForIdentity(identity)
if err != nil {
t.Fatalf("issue session %s: %v", role, err)
}
return session.Token
}

func TestNotificationListNotificationsEmptyOnFreshOrg(t *testing.T) {
t.Parallel()
setup := setupNotificationTest(t)
orgID := "00000000-0000-0000-0000-000000001001"
userID := "00000000-0000-0000-0000-000000001aa1"
token := issueNotificationSession(t, setup, userID, orgID, "owner-notif-fresh@example.com", "owner")

req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
req.Header.Set("Authorization", "Bearer "+token)
resp := httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if resp.Code != http.StatusOK {
t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
}
var body struct {
Notifications []any `json:"notifications"`
Unread        int   `json:"unread_count"`
}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode: %v", err)
}
if len(body.Notifications) != 0 {
t.Fatalf("expected empty notifications, got %d", len(body.Notifications))
}
if body.Unread != 0 {
t.Fatalf("expected unread=0, got %d", body.Unread)
}
}

func TestNotificationListAndMarkRead(t *testing.T) {
t.Parallel()
setup := setupNotificationTest(t)
orgID := "00000000-0000-0000-0000-000000001002"
userID := "00000000-0000-0000-0000-000000001aa2"
token := issueNotificationSession(t, setup, userID, orgID, "owner-notif-mark@example.com", "owner")

// Manually create a notification (simulating event hook)
ctx := context.WithValue(context.Background(), "auth_context", service.RequestAuthContext{
UserID:         userID,
OrganizationID: orgID,
Role:           "owner",
})

userIDPtr := userID
notif, err := setup.notificationService.CreateNotification(ctx, orgID, domain.NotificationKindWalletCredit, "Credit received", "You received 1000 credits", &userIDPtr, map[string]any{"amount": 1000})
if err != nil {
t.Fatalf("create notification: %v", err)
}

// List should show 1 unread
req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
req.Header.Set("Authorization", "Bearer "+token)
resp := httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if resp.Code != http.StatusOK {
t.Fatalf("list expected 200, got %d: %s", resp.Code, resp.Body.String())
}
var listBody struct {
Notifications []struct {
ID    string     `json:"id"`
Kind  string     `json:"kind"`
Title string     `json:"title"`
ReadAt *time.Time `json:"read_at"`
} `json:"notifications"`
Unread int `json:"unread_count"`
}
if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
t.Fatalf("decode list: %v", err)
}
if len(listBody.Notifications) != 1 {
t.Fatalf("expected 1 notification, got %d", len(listBody.Notifications))
}
if listBody.Unread != 1 {
t.Fatalf("expected unread=1, got %d", listBody.Unread)
}
if listBody.Notifications[0].ReadAt != nil {
t.Fatalf("expected unread notification (read_at=null), got %v", listBody.Notifications[0].ReadAt)
}

// Mark as read
markReq := bytes.NewBufferString(`{}`)
req = httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+notif.ID+":read", markReq)
req.Header.Set("Authorization", "Bearer "+token)
req.Header.Set("Content-Type", "application/json")
resp = httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if resp.Code != http.StatusOK {
t.Fatalf("mark read expected 200, got %d: %s", resp.Code, resp.Body.String())
}

// List again should show 0 unread
req = httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
req.Header.Set("Authorization", "Bearer "+token)
resp = httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
t.Fatalf("decode list after mark: %v", err)
}
if listBody.Unread != 0 {
t.Fatalf("expected unread=0 after mark read, got %d", listBody.Unread)
}
if listBody.Notifications[0].ReadAt == nil {
t.Fatalf("expected read_at to be set after mark read")
}
}

func TestNotificationMarkAllAsRead(t *testing.T) {
t.Parallel()
setup := setupNotificationTest(t)
orgID := "00000000-0000-0000-0000-000000001003"
userID := "00000000-0000-0000-0000-000000001aa3"
token := issueNotificationSession(t, setup, userID, orgID, "owner-notif-all@example.com", "owner")

// Create multiple notifications
ctx := context.WithValue(context.Background(), "auth_context", service.RequestAuthContext{
UserID:         userID,
OrganizationID: orgID,
Role:           "owner",
})

userIDPtr := userID
for i := 0; i < 3; i++ {
_, err := setup.notificationService.CreateNotification(ctx, orgID, domain.NotificationKindWalletCredit, "Credit", "Credit msg", &userIDPtr, nil)
if err != nil {
t.Fatalf("create notification %d: %v", i, err)
}
}

// Mark all as read
req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications:read-all", nil)
req.Header.Set("Authorization", "Bearer "+token)
req.Header.Set("Content-Type", "application/json")
resp := httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if resp.Code != http.StatusOK {
t.Fatalf("mark all read expected 200, got %d: %s", resp.Code, resp.Body.String())
}

// Unread count should be 0
req = httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
req.Header.Set("Authorization", "Bearer "+token)
resp = httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

var body struct {
Unread int `json:"unread_count"`
}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode: %v", err)
}
if body.Unread != 0 {
t.Fatalf("expected unread=0 after mark all, got %d", body.Unread)
}
}

func TestNotificationCrossOrgIsolation(t *testing.T) {
t.Parallel()
setup := setupNotificationTest(t)
orgID1 := "00000000-0000-0000-0000-000000001101"
orgID2 := "00000000-0000-0000-0000-000000001102"
user1ID := "00000000-0000-0000-0000-000000001bb1"
user2ID := "00000000-0000-0000-0000-000000001bb2"

token1 := issueNotificationSession(t, setup, user1ID, orgID1, "owner-notif-org1@example.com", "owner")
token2 := issueNotificationSession(t, setup, user2ID, orgID2, "owner-notif-org2@example.com", "owner")

// Create notification in org1
ctx1 := context.WithValue(context.Background(), "auth_context", service.RequestAuthContext{
UserID:         user1ID,
OrganizationID: orgID1,
Role:           "owner",
})

user1IDPtr := user1ID
_, err := setup.notificationService.CreateNotification(ctx1, orgID1, domain.NotificationKindWalletCredit, "Org1 Credit", "Credit in org1", &user1IDPtr, nil)
if err != nil {
t.Fatalf("create org1 notification: %v", err)
}

// User2 should not see org1 notification
req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
req.Header.Set("Authorization", "Bearer "+token2)
resp := httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if resp.Code != http.StatusOK {
t.Fatalf("org2 list expected 200, got %d", resp.Code)
}
var body struct {
Notifications []any `json:"notifications"`
Unread        int   `json:"unread_count"`
}
if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode: %v", err)
}
if len(body.Notifications) != 0 {
t.Fatalf("expected org2 to see 0 notifications, got %d", len(body.Notifications))
}
if body.Unread != 0 {
t.Fatalf("expected org2 unread=0, got %d", body.Unread)
}

// User1 should see the notification
req = httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
req.Header.Set("Authorization", "Bearer "+token1)
resp = httptest.NewRecorder()
setup.router.ServeHTTP(resp, req)

if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
t.Fatalf("decode: %v", err)
}
if len(body.Notifications) != 1 {
t.Fatalf("expected org1 to see 1 notification, got %d", len(body.Notifications))
}
}
