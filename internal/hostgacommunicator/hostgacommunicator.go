package hostgacommunicator

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/Azure/run-command-handler-linux/internal/requesthelper"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

const (
	hostGaPluginPort          = "32526"
	WireServerFallbackAddress = "http://168.63.129.16:32526"
)

// Interface for operations available when communicating with HostGAPlugin
type IHostGACommunicator interface {
	GetImmediateVMSettings(ctx *log.Context) (*VMSettings, error)
}

// HostGaCommunicator provides methods for retrieving VMSettings from the HostGAPlugin
type HostGACommunicator struct {
	vmRequestManager IVMSettingsRequestManager
}

func NewHostGACommunicator(requestManager IVMSettingsRequestManager) HostGACommunicator {
	return HostGACommunicator{vmRequestManager: requestManager}
}

type IVMSettingsRequestManager interface {
	GetVMSettingsRequestManager(ctx *log.Context) (*requesthelper.RequestManager, error)
}

// GetVMSettings returns the VMSettings for the current machine
func (c *HostGACommunicator) GetImmediateVMSettings(ctx *log.Context) (*VMSettings, error) {
	ctx.Log("message", "getting request manager")
	requestManager, err := c.vmRequestManager.GetVMSettingsRequestManager(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create the request manager")
	}

	ctx.Log("message", "attempting to make request with retries to retrieve VMSettings")
	resp, err := requesthelper.WithRetries(ctx, requestManager, requesthelper.ActualSleep)
	if err != nil {
		return nil, errors.Wrapf(err, "metadata request failed with retries.")
	}
	ctx.Log("message", "request completed. Reading body content from response")

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	ctx.Log("message", "attempting to parse VMSettings from json response")
	var vmSettings VMSettings
	if err := json.Unmarshal(body, &vmSettings); err != nil {
		return nil, errors.Wrapf(err, "failed to parse json")
	}

	ctx.Log("message", "VMSettings successfully parsed")
	return &vmSettings, nil
}

// Gets the URI to use to call the given operation name
func getOperationUri(ctx *log.Context, operationName string) (string, error) {
	// TODO: investigate why other extensions use the env var AZURE_GUEST_AGENT_WIRE_PROTOCOL_ADDRESS
	// and decide if we want to add that wire protocol address as a potential endpoint to use when provided
	ctx.Log("message", "creating uri to perform operation")
	uri, err := url.Parse(WireServerFallbackAddress)
	if err != nil {
		return "", errors.Wrap(err, "could not parse address "+WireServerFallbackAddress)
	}
	uri.Path = operationName
	return uri.String(), nil
}
