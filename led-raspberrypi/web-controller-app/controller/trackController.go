package controllers

import (
	"context"
	"encoding/json"
	"log"

	"github.com/kubeedge/examples/led-raspberrypi/web-controller-app/utils"
	devices "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"
	"github.com/astaxie/beego"
	"k8s.io/client-go/rest"
)

// DeviceStatus is used to patch device status
type DeviceStatus struct {
	Status devices.DeviceStatus `json:"status"`
}

// The device id of the counter
var deviceID = "led-light-instance-01"

// The default namespace in which the counter device instance resides
var namespace = "default"

// The default status of the counter
var originCmd = "OFF"

// The CRD client used to patch the device instance.
var crdClient *rest.RESTClient

// The twin value map
var statusMap = map[string]string{
	"ON":  "1",
	"OFF": "0",
}

func init() {
	// Create a client to talk to the K8S API server to patch the device CRDs
	kubeConfig, err := utils.KubeConfig()
	if err != nil {
		log.Fatalf("Failed to create KubeConfig, error : %v", err)
	}
	log.Println("Get kubeConfig successfully")

	crdClient, err = utils.NewCRDClient(kubeConfig)
	if err != nil {
		log.Fatalf("Failed to create device crd client , error : %v", err)
	}
	log.Println("Get crdClient successfully")
}

func UpdateStatus() map[string]string {
	result := DeviceStatus{}
	raw, _ := crdClient.Get().Namespace(namespace).Resource(utils.ResourceTypeDevices).Name(deviceID).DoRaw(context.TODO())

	status := map[string]string{
		"power-status": "OFF",
	}

	_ = json.Unmarshal(raw, &result)
	for _, twin := range result.Status.Twins {
		status["power-status"] = twin.Desired.Value
	}

	return status
}

// UpdateDeviceTwinWithDesiredTrack patches the desired state of
// the device twin with the command.
func UpdateDeviceTwinWithDesiredTrack(cmd string) bool {
	if cmd == originCmd {
		return true
	}

	status := buildStatusWithDesiredTrack(cmd)
	deviceStatus := &DeviceStatus{Status: status}
	body, err := json.Marshal(deviceStatus)
	if err != nil {
		log.Printf("Failed to marshal device status %v", deviceStatus)
		return false
	}
	result := crdClient.Patch(utils.MergePatchType).Namespace(namespace).Resource(utils.ResourceTypeDevices).Name(deviceID).Body(body).Do(context.TODO())
	if result.Error() != nil {
		log.Printf("Failed to patch device status %v of device %v in namespace %v \n error:%+v", deviceStatus, deviceID, namespace, result.Error())
		return false
	} else {
		log.Printf("Turn %s %s", cmd, deviceID)
	}
	originCmd = cmd

	return true
}

func buildStatusWithDesiredTrack(cmd string) devices.DeviceStatus {
	metadata := map[string]string{
		"type":      "string",
	}
	twins := []devices.Twin{{PropertyName: "power-status", Desired: devices.TwinProperty{Value: cmd, Metadata: metadata}}}
	devicestatus := devices.DeviceStatus{Twins: twins}
	return devicestatus
}

type TrackController struct {
	beego.Controller
}

// Index is the initial view
func (controller *TrackController) Index() {
	log.Println("Index Start")

	controller.Layout = "layout.html"
	controller.TplName = "content.html"
	controller.LayoutSections = map[string]string{}
	controller.LayoutSections["PageHead"] = "head.html"

	log.Println("Index Finish")
}

// Control
func (controller *TrackController) ControlTrack() {
	// Get track id
	params := struct {
		TrackID string `form:":trackId"`
	}{controller.GetString(":trackId")}

	resultCode := 0

	status := map[string]string{}

	log.Printf("ControlTrack: %s", params.TrackID)
	// update track
	if params.TrackID == "ON" {
		UpdateDeviceTwinWithDesiredTrack(params.TrackID)
		resultCode = 1
	} else if params.TrackID == "OFF" {
		UpdateDeviceTwinWithDesiredTrack(params.TrackID)
		resultCode = 2
	} 

	// response
	controller.AjaxResponse(resultCode, status, nil)
}

// AjaxResponse returns a standard ajax response.
func (Controller *TrackController) AjaxResponse(resultCode int, resultString map[string]string, data interface{}) {
	response := struct {
		Result       int
		ResultString map[string]string
		ResultObject interface{}
	}{
		Result:       resultCode,
		ResultString: resultString,
		ResultObject: data,
	}

	Controller.Data["json"] = response
	Controller.ServeJSON()
}
