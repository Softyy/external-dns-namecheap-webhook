package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOpenMetricsInstance(t *testing.T) {
	m := GetOpenMetricsInstance()
	assert.NotNil(t, m)
	assert.NotNil(t, m.GetRegistry())

	m2 := GetOpenMetricsInstance()
	assert.Equal(t, m, m2)
}

func TestIncSuccessfulApiCallsTotal(t *testing.T) {
	m := GetOpenMetricsInstance()
	m.IncSuccessfulApiCallsTotal("test_action")
}

func TestIncFailedApiCallsTotal(t *testing.T) {
	m := GetOpenMetricsInstance()
	m.IncFailedApiCallsTotal("test_action")
}

func TestSetFilteredOutZones(t *testing.T) {
	m := GetOpenMetricsInstance()
	m.SetFilteredOutZones(5)
}

func TestSetSkippedRecords(t *testing.T) {
	m := GetOpenMetricsInstance()
	m.SetSkippedRecords("example.com", 3)
}

func TestAddApiDelayHist(t *testing.T) {
	m := GetOpenMetricsInstance()
	m.AddApiDelayHist("test_action", 150)
}

func TestGetHandler(t *testing.T) {
	m := GetOpenMetricsInstance()
	handler := m.GetHandler()
	assert.NotNil(t, handler)

	m.IncSuccessfulApiCallsTotal("get_zones")

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}