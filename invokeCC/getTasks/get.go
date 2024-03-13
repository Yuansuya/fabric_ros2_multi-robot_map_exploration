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
	"github.com/TIERS/rclgo/pkg/rclgo"
	"encoding/json"
) 

var cli *channel.Client 
var rclNode *rclgo.Node
type FrontierPoint struct{
	X float64 `json:"x"`
	Y float64 `json:"y"`
	PreNodeID int `json:"preID"`
	DisFromPreNode float64 `json:"distance"`
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

	rclNode, rclErr = rclContext.NewNode("get_task_node", "")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	sub, subErr := rclNode.NewSubscription("get_all_frontier", std_msgs.BoolTypeSupport, getTaskCallback)
	if subErr != nil {
		log.Fatalf("Unable to create publisher: %v", subErr)
	}


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Spin(ctx, 100)
	defer sub.Close()
	<-doneChannel

}


func getTaskCallback(subscription *rclgo.Subscription){
	fmt.Println("getTaskCallback")
	msg := std_msgs.NewBool()
    

    _, err := subscription.TakeMessage(msg)
	if err != nil {
		fmt.Println("Failed to take message:", err)
		return
	}
	//commandded
	arg := `{"Args":["GetTask"]}`

	//call cmd
	cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tacc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
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

	var taskPoints []FrontierPoint
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
		err := json.Unmarshal([]byte(payloadStr), &taskPoints)
		if err != nil {
			fmt.Println("unmarsal json waypoint failed")
			return 
		}
	}

	fmt.Println("filter_outputPoints:  ", taskPoints)

	//pub task to visual 
	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, pubErr := rclNode.NewPublisher("all_frontier_visualization", geo_msgs.PoseArrayTypeSupport, opts)
	if pubErr != nil {
		log.Fatalf("Unable to create publisher: %v", pubErr)
	}

	all_frontier_msg := geo_msgs.NewPoseArray()

	for _, taskPoint := range(taskPoints){
		pt := geo_msgs.NewPose()
		pt.Position.X = taskPoint.X
		pt.Position.Y = taskPoint.Y
		all_frontier_msg.Poses = append(all_frontier_msg.Poses, *pt)
	}
	pub.Publish(all_frontier_msg)
	fmt.Println("published taskPoints")
	return 

}

