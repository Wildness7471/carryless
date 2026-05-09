package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestHandleTrips(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	w := doJSONRequest(router, http.MethodGet, "/trips", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateTrip(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{
		"name":        {"Summer Hike 2025"},
		"location":    {"Yosemite"},
		"description": {"A great trip"},
		"start_date":  {"2025-07-01"},
		"end_date":    {"2025-07-07"},
	}
	w := doRequest(router, http.MethodPost, "/trips",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after create, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM trips WHERE user_id = ? AND name = 'Summer Hike 2025'", user.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 trip, got %d", count)
	}
}

func TestHandleTripDetail(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{"name": {"Test Trip"}, "start_date": {"2025-08-01"}, "end_date": {"2025-08-05"}}
	doRequest(router, http.MethodPost, "/trips", form.Encode(), "application/x-www-form-urlencoded", cookie)

	var tripID string
	db.QueryRow("SELECT id FROM trips WHERE user_id = ? AND name = 'Test Trip'", user.ID).Scan(&tripID)

	w := doJSONRequest(router, http.MethodGet, "/trips/"+tripID, "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleUpdateTrip(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{"name": {"Original Trip"}, "start_date": {"2025-09-01"}, "end_date": {"2025-09-05"}}
	doRequest(router, http.MethodPost, "/trips", form.Encode(), "application/x-www-form-urlencoded", cookie)

	var tripID string
	db.QueryRow("SELECT id FROM trips WHERE user_id = ? AND name = 'Original Trip'", user.ID).Scan(&tripID)

	updateForm := url.Values{"name": {"Updated Trip"}, "start_date": {"2025-09-01"}, "end_date": {"2025-09-07"}}
	w := doRequest(router, http.MethodPost, "/trips/" + tripID,
		updateForm.Encode(), "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after update, got %d: %s", w.Code, w.Body.String())
	}

	var name string
	db.QueryRow("SELECT name FROM trips WHERE id = ?", tripID).Scan(&name)
	if name != "Updated Trip" {
		t.Errorf("trip name not updated, got %q", name)
	}
}

func TestHandleDeleteTrip(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{"name": {"ToDelete Trip"}, "start_date": {"2025-10-01"}, "end_date": {"2025-10-05"}}
	doRequest(router, http.MethodPost, "/trips", form.Encode(), "application/x-www-form-urlencoded", cookie)

	var tripID string
	db.QueryRow("SELECT id FROM trips WHERE user_id = ? AND name = 'ToDelete Trip'", user.ID).Scan(&tripID)

	w := doRequest(router, http.MethodPost, "/trips/"+tripID+"/delete",
		"", "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after delete, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM trips WHERE id = ?", tripID).Scan(&count)
	if count != 0 {
		t.Errorf("trip still exists after delete")
	}
}

func TestHandleTripChecklist(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	tripForm := url.Values{"name": {"Checklist Trip"}, "start_date": {"2025-11-01"}, "end_date": {"2025-11-05"}}
	doRequest(router, http.MethodPost, "/trips", tripForm.Encode(), "application/x-www-form-urlencoded", cookie)

	var tripID string
	db.QueryRow("SELECT id FROM trips WHERE user_id = ?", user.ID).Scan(&tripID)

	// Add checklist item (endpoint expects JSON body)
	wAdd := doRequest(router, http.MethodPost,
		"/trips/"+tripID+"/checklist",
		`{"content":"Pack rain jacket"}`, "application/json", cookie)
	if wAdd.Code != http.StatusOK && wAdd.Code != http.StatusCreated {
		t.Fatalf("expected success adding checklist item, got %d: %s", wAdd.Code, wAdd.Body.String())
	}

	var itemID int
	db.QueryRow("SELECT id FROM trip_checklist_items WHERE trip_id = ?", tripID).Scan(&itemID)
	if itemID == 0 {
		t.Fatal("no checklist item created")
	}

	// Toggle item
	toggleForm := url.Values{}
	wToggle := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/trips/%s/checklist/%d/toggle", tripID, itemID), toggleForm.Encode(), cookie)
	if wToggle.Code != http.StatusOK {
		t.Errorf("expected 200 toggling checklist item, got %d: %s", wToggle.Code, wToggle.Body.String())
	}

	// Delete checklist item
	wDelete := doJSONRequest(router, http.MethodDelete,
		fmt.Sprintf("/trips/%s/checklist/%d", tripID, itemID), "", cookie)
	if wDelete.Code != http.StatusOK && wDelete.Code != http.StatusNoContent {
		t.Errorf("expected success deleting checklist item, got %d: %s", wDelete.Code, wDelete.Body.String())
	}
}

func TestHandleTripTransport(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	tripForm := url.Values{"name": {"Transport Trip"}, "start_date": {"2025-12-01"}, "end_date": {"2025-12-05"}}
	doRequest(router, http.MethodPost, "/trips", tripForm.Encode(), "application/x-www-form-urlencoded", cookie)

	var tripID string
	db.QueryRow("SELECT id FROM trips WHERE user_id = ?", user.ID).Scan(&tripID)

	// Add transport step (endpoint expects JSON body)
	transportJSON := `{"journey_type":"outbound","departure_place":"Home","arrival_place":"Trailhead","transport_type":"train","departure_datetime":"2025-12-01T08:00","arrival_datetime":"2025-12-01T12:00"}`
	wAdd := doRequest(router, http.MethodPost,
		"/trips/"+tripID+"/transport",
		transportJSON, "application/json", cookie)
	if wAdd.Code != http.StatusOK && wAdd.Code != http.StatusCreated {
		t.Fatalf("expected success adding transport step, got %d: %s", wAdd.Code, wAdd.Body.String())
	}

	var stepResp map[string]interface{}
	json.NewDecoder(wAdd.Body).Decode(&stepResp)

	var stepID int
	db.QueryRow("SELECT id FROM trip_transport_steps WHERE trip_id = ?", tripID).Scan(&stepID)
	if stepID == 0 {
		t.Fatal("no transport step created")
	}

	// Delete transport step
	wDelete := doJSONRequest(router, http.MethodDelete,
		fmt.Sprintf("/trips/%s/transport/%d", tripID, stepID), "", cookie)
	if wDelete.Code != http.StatusOK && wDelete.Code != http.StatusNoContent {
		t.Errorf("expected success deleting transport step, got %d: %s", wDelete.Code, wDelete.Body.String())
	}
}
