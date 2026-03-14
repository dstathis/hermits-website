package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		fmt.Println("TEST_DATABASE_URL not set, skipping db tests")
		os.Exit(0)
	}

	var err error
	testDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open test database: %v", err)
	}

	// Wait for DB to be ready
	for i := 0; i < 30; i++ {
		if err := testDB.Ping(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := testDB.Ping(); err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}

func cleanTable(t *testing.T, table string) {
	t.Helper()
	_, err := testDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	if err != nil {
		t.Fatalf("failed to clean %s: %v", table, err)
	}
}

// --- Events ---

func TestCreateAndGetEvent(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")
	cleanTable(t, "subscribers")
	cleanTable(t, "events")

	e := &Event{
		Title:       "Legacy Tournament",
		Format:      "Legacy",
		Description: "Proxy-friendly legacy event",
		Date:        time.Now().Add(24 * time.Hour),
		Location:    "Dragonphoenix Inn",
		LocationURL: "https://maps.example.com",
		EntryFee:    "5€",
	}

	if err := CreateEvent(testDB, e); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	if e.ID == "" {
		t.Fatal("expected event ID to be set")
	}
	if e.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	got, err := GetEventByID(testDB, e.ID)
	if err != nil {
		t.Fatalf("GetEventByID: %v", err)
	}
	if got.Title != "Legacy Tournament" {
		t.Errorf("expected title %q, got %q", "Legacy Tournament", got.Title)
	}
	if got.LocationURL != "https://maps.example.com" {
		t.Errorf("expected location URL, got %q", got.LocationURL)
	}
}

func TestGetEventByID_NotFound(t *testing.T) {
	got, err := GetEventByID(testDB, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent event")
	}
}

func TestUpdateEvent(t *testing.T) {
	cleanTable(t, "events")

	e := &Event{
		Title:  "Original",
		Format: "Legacy",
		Date:   time.Now().Add(24 * time.Hour),
	}
	if err := CreateEvent(testDB, e); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	e.Title = "Updated"
	if err := UpdateEvent(testDB, e); err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}

	got, _ := GetEventByID(testDB, e.ID)
	if got.Title != "Updated" {
		t.Errorf("expected updated title, got %q", got.Title)
	}
}

func TestDeleteEvent(t *testing.T) {
	cleanTable(t, "events")

	e := &Event{Title: "To Delete", Format: "Legacy", Date: time.Now().Add(24 * time.Hour)}
	if err := CreateEvent(testDB, e); err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}

	if err := DeleteEvent(testDB, e.ID); err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}

	got, _ := GetEventByID(testDB, e.ID)
	if got != nil {
		t.Error("expected event to be deleted")
	}
}

func TestGetUpcomingAndPastEvents(t *testing.T) {
	cleanTable(t, "events")

	future := &Event{Title: "Future", Format: "Legacy", Date: time.Now().Add(48 * time.Hour)}
	past := &Event{Title: "Past", Format: "Premodern", Date: time.Now().Add(-48 * time.Hour)}
	CreateEvent(testDB, future)
	CreateEvent(testDB, past)

	upcoming, err := GetUpcomingEvents(testDB)
	if err != nil {
		t.Fatalf("GetUpcomingEvents: %v", err)
	}
	if len(upcoming) != 1 || upcoming[0].Title != "Future" {
		t.Errorf("expected 1 upcoming event, got %d", len(upcoming))
	}

	pastEvents, err := GetPastEvents(testDB)
	if err != nil {
		t.Fatalf("GetPastEvents: %v", err)
	}
	if len(pastEvents) != 1 || pastEvents[0].Title != "Past" {
		t.Errorf("expected 1 past event, got %d", len(pastEvents))
	}
}

func TestGetNextEvent(t *testing.T) {
	cleanTable(t, "events")

	e1 := &Event{Title: "Later", Format: "Legacy", Date: time.Now().Add(72 * time.Hour)}
	e2 := &Event{Title: "Sooner", Format: "Legacy", Date: time.Now().Add(24 * time.Hour)}
	CreateEvent(testDB, e1)
	CreateEvent(testDB, e2)

	next, err := GetNextEvent(testDB)
	if err != nil {
		t.Fatalf("GetNextEvent: %v", err)
	}
	if next == nil {
		t.Fatal("expected next event")
	}
	if next.Title != "Sooner" {
		t.Errorf("expected 'Sooner', got %q", next.Title)
	}
}

func TestGetNextEvent_None(t *testing.T) {
	cleanTable(t, "events")

	next, err := GetNextEvent(testDB)
	if err != nil {
		t.Fatalf("GetNextEvent: %v", err)
	}
	if next != nil {
		t.Error("expected nil when no upcoming events")
	}
}

func TestGetAllEvents(t *testing.T) {
	cleanTable(t, "events")

	CreateEvent(testDB, &Event{Title: "A", Format: "Legacy", Date: time.Now().Add(24 * time.Hour)})
	CreateEvent(testDB, &Event{Title: "B", Format: "Legacy", Date: time.Now().Add(-24 * time.Hour)})

	all, err := GetAllEvents(testDB)
	if err != nil {
		t.Fatalf("GetAllEvents: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 events, got %d", len(all))
	}
}

// --- Subscribers ---

func TestCreateSubscriber(t *testing.T) {
	cleanTable(t, "subscribers")

	sub, err := CreateSubscriber(testDB, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("CreateSubscriber: %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscriber to be returned")
	}
	if sub.Email != "test@example.com" {
		t.Errorf("expected email %q, got %q", "test@example.com", sub.Email)
	}
	if sub.Confirmed {
		t.Error("expected subscriber to be unconfirmed initially")
	}
	if sub.Token == "" {
		t.Error("expected subscriber token to be set")
	}
}

func TestCreateSubscriber_Duplicate(t *testing.T) {
	cleanTable(t, "subscribers")

	CreateSubscriber(testDB, "dupe@example.com", "First")
	sub, err := CreateSubscriber(testDB, "dupe@example.com", "Second")
	if err != nil {
		t.Fatalf("expected no error for duplicate, got: %v", err)
	}
	if sub != nil {
		t.Error("expected nil return for duplicate email")
	}
}

func TestConfirmSubscriber(t *testing.T) {
	cleanTable(t, "subscribers")

	sub, _ := CreateSubscriber(testDB, "confirm@example.com", "")
	if err := ConfirmSubscriber(testDB, sub.Token); err != nil {
		t.Fatalf("ConfirmSubscriber: %v", err)
	}

	// Should fail on second confirm (already confirmed)
	if err := ConfirmSubscriber(testDB, sub.Token); err == nil {
		t.Error("expected error on second confirm")
	}
}

func TestUnsubscribeByToken(t *testing.T) {
	cleanTable(t, "subscribers")

	sub, _ := CreateSubscriber(testDB, "unsub@example.com", "")
	ConfirmSubscriber(testDB, sub.Token)

	if err := UnsubscribeByToken(testDB, sub.Token); err != nil {
		t.Fatalf("UnsubscribeByToken: %v", err)
	}

	// Verify the subscriber is now unconfirmed
	subs, _ := GetConfirmedSubscriberEmails(testDB)
	for _, s := range subs {
		if s.Email == "unsub@example.com" {
			t.Error("expected subscriber to be unconfirmed after unsubscribe")
		}
	}
}

func TestUnsubscribeByToken_InvalidToken(t *testing.T) {
	if err := UnsubscribeByToken(testDB, "nonexistent-token"); err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestGetConfirmedSubscriberEmails(t *testing.T) {
	cleanTable(t, "subscribers")

	sub1, _ := CreateSubscriber(testDB, "a@example.com", "A")
	CreateSubscriber(testDB, "b@example.com", "B") // unconfirmed
	ConfirmSubscriber(testDB, sub1.Token)

	subs, err := GetConfirmedSubscriberEmails(testDB)
	if err != nil {
		t.Fatalf("GetConfirmedSubscriberEmails: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("expected 1 confirmed subscriber, got %d", len(subs))
	}
	if subs[0].Email != "a@example.com" {
		t.Errorf("expected a@example.com, got %q", subs[0].Email)
	}
}

func TestGetAllSubscribers(t *testing.T) {
	cleanTable(t, "subscribers")

	CreateSubscriber(testDB, "x@example.com", "X")
	CreateSubscriber(testDB, "y@example.com", "Y")

	all, err := GetAllSubscribers(testDB)
	if err != nil {
		t.Fatalf("GetAllSubscribers: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 subscribers, got %d", len(all))
	}
}

func TestDeleteSubscriber(t *testing.T) {
	cleanTable(t, "subscribers")

	sub, _ := CreateSubscriber(testDB, "delete@example.com", "")
	if err := DeleteSubscriber(testDB, sub.ID); err != nil {
		t.Fatalf("DeleteSubscriber: %v", err)
	}

	all, _ := GetAllSubscribers(testDB)
	if len(all) != 0 {
		t.Errorf("expected 0 subscribers after delete, got %d", len(all))
	}
}

// --- Admin Users ---

func TestCreateAdminUser(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	if err := CreateAdminUser(testDB, "admin", "password123"); err != nil {
		t.Fatalf("CreateAdminUser: %v", err)
	}

	// Duplicate should error
	if err := CreateAdminUser(testDB, "admin", "other"); err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestAuthenticate(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "testadmin", "correct-password")

	user, err := Authenticate(testDB, "testadmin", "correct-password")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user == nil {
		t.Fatal("expected user to be returned")
	}
	if user.Username != "testadmin" {
		t.Errorf("expected username 'testadmin', got %q", user.Username)
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "admin2", "correct")

	user, err := Authenticate(testDB, "admin2", "wrong")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Error("expected nil for wrong password")
	}
}

func TestAuthenticate_NonExistent(t *testing.T) {
	user, err := Authenticate(testDB, "nonexistent", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestInviteAndAccept(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	token, err := InviteAdminUser(testDB, "invitee", "invite@example.com")
	if err != nil {
		t.Fatalf("InviteAdminUser: %v", err)
	}
	if token == "" {
		t.Fatal("expected invite token")
	}

	// Pending invitee can't log in
	user, _ := Authenticate(testDB, "invitee", "anything")
	if user != nil {
		t.Error("pending invite should not authenticate")
	}

	// Look up by invite token
	invited, err := GetAdminUserByInviteToken(testDB, token)
	if err != nil {
		t.Fatalf("GetAdminUserByInviteToken: %v", err)
	}
	if invited == nil || invited.Username != "invitee" {
		t.Fatal("expected to find invited user")
	}

	// Accept invite
	if err := AcceptInvite(testDB, token, "newpassword"); err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}

	// Now can log in
	user, _ = Authenticate(testDB, "invitee", "newpassword")
	if user == nil {
		t.Error("expected to authenticate after accepting invite")
	}

	// Token should be cleared
	invited, _ = GetAdminUserByInviteToken(testDB, token)
	if invited != nil {
		t.Error("expected invite token to be cleared after accept")
	}
}

func TestChangePassword(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "changepw", "oldpass12")
	user, _ := Authenticate(testDB, "changepw", "oldpass12")

	if err := ChangePassword(testDB, user.ID, "oldpass12", "newpass12"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	// Old password shouldn't work
	u, _ := Authenticate(testDB, "changepw", "oldpass12")
	if u != nil {
		t.Error("old password should not work after change")
	}

	// New password should work
	u, _ = Authenticate(testDB, "changepw", "newpass12")
	if u == nil {
		t.Error("new password should work after change")
	}
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "changepw2", "oldpass12")
	user, _ := Authenticate(testDB, "changepw2", "oldpass12")

	if err := ChangePassword(testDB, user.ID, "wrongcurrent", "newpass12"); err == nil {
		t.Error("expected error for wrong current password")
	}
}

// --- Sessions ---

func TestCreateAndGetSession(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "sessuser", "pass1234")
	user, _ := Authenticate(testDB, "sessuser", "pass1234")

	session, err := CreateSession(testDB, user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected session ID")
	}

	got, err := GetSession(testDB, session.ID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got == nil {
		t.Fatal("expected to find session")
	}
	if got.UserID != user.ID {
		t.Errorf("expected user ID %q, got %q", user.ID, got.UserID)
	}
}

func TestDeleteSession(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "sessuser2", "pass1234")
	user, _ := Authenticate(testDB, "sessuser2", "pass1234")
	session, _ := CreateSession(testDB, user.ID)

	if err := DeleteSession(testDB, session.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	got, _ := GetSession(testDB, session.ID)
	if got != nil {
		t.Error("expected session to be deleted")
	}
}

func TestGetSession_NonExistent(t *testing.T) {
	got, err := GetSession(testDB, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent session")
	}
}

func TestGetAllAdminUsers(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "user1", "pass1234")
	CreateAdminUser(testDB, "user2", "pass5678")

	users, err := GetAllAdminUsers(testDB)
	if err != nil {
		t.Fatalf("GetAllAdminUsers: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 admin users, got %d", len(users))
	}
}

func TestDeleteAdminUser(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "todelete", "pass1234")
	user, _ := Authenticate(testDB, "todelete", "pass1234")

	if err := DeleteAdminUser(testDB, user.ID); err != nil {
		t.Fatalf("DeleteAdminUser: %v", err)
	}

	users, _ := GetAllAdminUsers(testDB)
	if len(users) != 0 {
		t.Errorf("expected 0 users after delete, got %d", len(users))
	}
}

func TestDeleteAdminUser_CascadesSessions(t *testing.T) {
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")

	CreateAdminUser(testDB, "cascadeuser", "pass1234")
	user, _ := Authenticate(testDB, "cascadeuser", "pass1234")
	session, _ := CreateSession(testDB, user.ID)

	DeleteAdminUser(testDB, user.ID)

	got, _ := GetSession(testDB, session.ID)
	if got != nil {
		t.Error("expected session to be cascade-deleted with user")
	}
}
