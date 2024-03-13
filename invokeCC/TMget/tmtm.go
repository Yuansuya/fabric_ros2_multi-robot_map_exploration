package main

import(
	"fmt"
	"log"
	"sync"
	"context"
    "os/exec"
	"regexp"
	"strings"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	geo_msgs "github.com/TIERS/rclgo-msgs/geometry_msgs/msg"
	std_msgs "github.com/TIERS/rclgo-msgs/std_msgs/msg"
	vis_msgs "github.com/TIERS/rclgo-msgs/visualization_msgs/msg"
	"github.com/TIERS/rclgo/pkg/rclgo"
	"encoding/json"
) 

var cli *channel.Client 
var rclNode *rclgo.Node
type Point struct{
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TM struct{
	Node []Point `json:"node"`
	Edge [][]int `json:"edge"`
	RobotCurrentLocationNode map[string]int `json:"robotnode"`
}

func main(){

	
	var doneChannel = make(chan bool)
	var wg sync.WaitGroup
	rclArgs, rclErr := rclgo.NewRCLArgs("")
	if rclErr != nil {
		log.Fatal(rclErr)
	}
	
	rclContext, rclErr := rclgo.NewContext(&wg, 0, rclArgs)
	if rclErr != nil {
		log.Fatal(rclErr)
	}
	defer rclContext.Close()

	rclNode, rclErr = rclContext.NewNode("getTM_node", "")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	sub, subErr := rclNode.NewSubscription("get_TM", std_msgs.BoolTypeSupport, getTMCallback)
	if subErr != nil {
		log.Fatalf("Unable to create publisher: %v", subErr)
	}


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Spin(ctx, 100)
	defer sub.Close()
	<-doneChannel

}


func getTMCallback(subscription *rclgo.Subscription){
	fmt.Println("getTMCallback")
	msg := std_msgs.NewBool()
    

    _, err := subscription.TakeMessage(msg)
	if err != nil {
		fmt.Println("Failed to take message:", err)
		return
	}
	arg := `{"Args":["GetTopoMap"]}`
	

	//call cmd
	cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tmcc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
	fmt.Println("cmd done")

	//out
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error executing command:", err)
		// fmt.Println(string(output))
		return
	}

	str_output := string(output)
	//fetch status number 
	re := regexp.MustCompile(`status:(\d+)`)
	matches := re.FindStringSubmatch(str_output)
	var status string
	if len(matches) > 1 {
		status = matches[1]
	}
	fmt.Println("got status number: ", status)
	if status != "200"{
		fmt.Println("Chaincode failed ...")
		return 
	}

	var tm TM
	if strings.Contains(str_output, "payload") {
		// 取得payload欄位的值
		payloadStartIndex := strings.Index(str_output, "payload:") + len("payload:")
		payloadStr := str_output[payloadStartIndex:]
		// fmt.Println(payloadStr)
		//trim "" and remove \\
		payloadStr = strings.TrimSpace(payloadStr)
		payloadStr = strings.Trim(payloadStr, "\"")
		payloadStr = strings.ReplaceAll(payloadStr, "\\", "")
		// fmt.Println(payloadStr)
		// deal with payload 
		err := json.Unmarshal([]byte(payloadStr), &tm)
		if err != nil {
			fmt.Println("unmarsal json waypoint failed")
			return 
		}
	}

	fmt.Println("tm:  ", tm)

	//maker_array
	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, pubErr := rclNode.NewPublisher("visualization_markers", vis_msgs.MarkerArrayTypeSupport, opts)
	if pubErr != nil {
		log.Fatalf("Unable to create publisher: %v", pubErr)
	}

	marker_array := vis_msgs.NewMarkerArray()

	pointMarker := vis_msgs.NewMarker()
	pointMarker.Action = 0
	pointMarker.Type = 7
	pointMarker.Header.FrameId= "map"
	pointMarker.Id = 0
	pointMarker.Scale.X = 0.2  
	pointMarker.Scale.Y = 0.2
	pointMarker.Scale.Z = 0.2
	pointMarker.Color.R = 1.0  
	pointMarker.Color.G = 0.0
	pointMarker.Color.B = 0.0
	pointMarker.Color.A = 1.0
	pointMarker.Points = make([]geo_msgs.Point, len(tm.Node))

	for i:=0 ; i< len(tm.Node); i++{
		pt := geo_msgs.NewPoint()
		pt.X= tm.Node[i].X* 1.0
		pt.Y= tm.Node[i].Y* 1.0
		pointMarker.Points= append(pointMarker.Points, *pt) 
	}
	marker_array.Markers= append(marker_array.Markers, *pointMarker)
	
	edgeMarker := vis_msgs.NewMarker()
	edgeMarker.Action = 0
	edgeMarker.Type = 5
	edgeMarker.Header.FrameId= "map"
	edgeMarker.Id = 1
	edgeMarker.Scale.X = 0.1  
	edgeMarker.Color.R = 0.0  
	edgeMarker.Color.G = 0.0
	edgeMarker.Color.B = 1.0
	edgeMarker.Color.A = 1.0
	edgeMarker.Points = make([]geo_msgs.Point, 0)

	for _, edge := range tm.Edge {
		start := tm.Node[edge[0]]
		end := tm.Node[edge[1]]
		edgeMarker.Points = append(edgeMarker.Points, geo_msgs.Point{X: start.X, Y: start.Y, Z: 0.0})
		edgeMarker.Points = append(edgeMarker.Points, geo_msgs.Point{X: end.X, Y: end.Y, Z: 0.0})
	}
		
	marker_array.Markers= append(marker_array.Markers, *edgeMarker)

	pub.Publish(marker_array)


}

