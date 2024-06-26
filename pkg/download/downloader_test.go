package download_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/azure-extension-foundation/msi"
	"github.com/Azure/run-command-handler-linux/pkg/download"
	"github.com/ahmetalpbalkan/go-httpbin"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

type badDownloader struct{ calls int }

var (
	testctx = log.NewContext(log.NewNopLogger())
)

func (b *badDownloader) GetRequest() (*http.Request, error) {
	b.calls++
	return nil, errors.New("expected error")
}

func TestDownload_wrapsGetRequestError(t *testing.T) {
	_, _, err := download.Download(testctx, new(badDownloader))
	require.NotNil(t, err)
	require.EqualError(t, err, "failed to create http request: expected error")
}

func TestDownload_wrapsHTTPError(t *testing.T) {
	_, _, err := download.Download(testctx, download.NewURLDownload("bad url"))
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "http request failed:")
}

func TestDownload_wrapsCommonErrorCodes(t *testing.T) {
	srv := httptest.NewServer(httpbin.GetMux())
	defer srv.Close()

	for _, code := range []int{
		http.StatusNotFound,
		http.StatusForbidden,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusBadRequest,
		http.StatusUnauthorized,
	} {
		respCode, _, err := download.Download(testctx, download.NewURLDownload(fmt.Sprintf("%s/status/%d", srv.URL, code)))
		require.NotNil(t, err, "not failed for code:%d", code)
		require.Equal(t, code, respCode)
		switch respCode {
		case http.StatusNotFound:
			require.Contains(t, err.Error(), "because it does not exist")
		case http.StatusForbidden:
			require.Contains(t, err.Error(), "Please verify the machine has network connectivity")
		case http.StatusInternalServerError:
			require.Contains(t, err.Error(), "due to an issue with storage")
		case http.StatusBadRequest:
			require.Contains(t, err.Error(), "because parts of the request were incorrectly formatted, missing, and/or invalid")
		case http.StatusUnauthorized:
			require.Contains(t, err.Error(), "because access was denied")
		}
	}
}

func TestDownload_statusOKSucceeds(t *testing.T) {
	srv := httptest.NewServer(httpbin.GetMux())
	defer srv.Close()

	_, body, err := download.Download(testctx, download.NewURLDownload(srv.URL+"/status/200"))
	require.Nil(t, err)
	defer body.Close()
	require.NotNil(t, body)
}

func TestDowload_msiDownloaderErrorMessage(t *testing.T) {
	var mockMsiProvider download.MsiProvider = func() (msi.Msi, error) {
		return msi.Msi{AccessToken: "fakeAccessToken"}, nil
	}
	srv := httptest.NewServer(httpbin.GetMux())
	defer srv.Close()

	msiDownloader404 := download.NewBlobWithMsiDownload(srv.URL+"/status/404", mockMsiProvider)

	returnCode, body, err := download.Download(testctx, msiDownloader404)
	require.True(t, strings.Contains(err.Error(), download.MsiDownload404ErrorString), "error string doesn't contain the correct message")
	require.Contains(t, err.Error(), "For more information, see https://aka.ms/RunCommandManagedLinux", "error string doesn't contain full message")
	require.Nil(t, body, "body is not nil for failed download")
	require.Equal(t, 404, returnCode, "return code was not 404")

	msiDownloader403 := download.NewBlobWithMsiDownload(srv.URL+"/status/403", mockMsiProvider)
	returnCode, body, err = download.Download(testctx, msiDownloader403)
	require.True(t, strings.Contains(err.Error(), download.MsiDownload403ErrorString), "error string doesn't contain the correct message")
	require.Contains(t, err.Error(), "For more information, see https://aka.ms/RunCommandManagedLinux", "error string doesn't contain full message")
	require.Nil(t, body, "body is not nil for failed download")
	require.Equal(t, 403, returnCode, "return code was not 403")

	// Should use default error message for any error code other than 400, 401, 403, 404, and 409
	msiDownloader500 := download.NewBlobWithMsiDownload(srv.URL+"/status/500", mockMsiProvider)
	returnCode, body, err = download.Download(testctx, msiDownloader500)
	fmt.Println(err.Error())
	require.Contains(t, err.Error(), "Use either a public script URI that points to .sh file", "error string doesn't contain full message")
	require.Contains(t, err.Error(), "For more information, see https://aka.ms/RunCommandManagedLinux", "error string doesn't contain full message")
	require.Nil(t, body, "body is not nil for failed download")
	require.Equal(t, 500, returnCode, "return code was not 500")

}

func TestDownload_retrievesBody(t *testing.T) {
	srv := httptest.NewServer(httpbin.GetMux())
	defer srv.Close()

	_, body, err := download.Download(testctx, download.NewURLDownload(srv.URL+"/bytes/65536"))
	require.Nil(t, err)
	defer body.Close()
	b, err := ioutil.ReadAll(body)
	require.Nil(t, err)
	require.EqualValues(t, 65536, len(b))
}

func TestDownload_bodyClosesWithoutError(t *testing.T) {
	srv := httptest.NewServer(httpbin.GetMux())
	defer srv.Close()

	_, body, err := download.Download(testctx, download.NewURLDownload(srv.URL+"/get"))
	require.Nil(t, err)
	require.Nil(t, body.Close(), "body should close fine")
}
