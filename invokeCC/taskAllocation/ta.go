package main

import(
	"fmt"
	"log"
	"sync"
	"context"
	"strconv"
	"strings"
	"os/exec"
	"regexp"
    "encoding/json"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	geo_msgs "github.com/TIERS/rclgo-msgs/geometry_msgs/msg"
	std_msgs "github.com/TIERS/rclgo-msgs/std_msgs/msg"
	"github.com/TIERS/rclgo/pkg/rclgo"
) 

var cli *channel.Client 
var rclNode *rclgo.Node
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
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

	rclNode, rclErr = rclContext.NewNode("ta", "")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	sub, subErr := rclNode.NewSubscription("ta_service", std_msgs.UInt32TypeSupport, taCallback)
	if subErr != nil {
		log.Fatalf("Unable to create publisher: %v", subErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Spin(ctx, 10)
	defer sub.Close()
	<-doneChannel

}


func taCallback(subscription *rclgo.Subscription){
	fmt.Println("TAcallback")
	msg := std_msgs.NewUInt32()
    

    _, err := subscription.TakeMessage(msg)

	if err != nil {
		fmt.Println("Failed to take message:", err)
		return
	}
	robotID := msg.Data >> 16   
    robot_stand_node := msg.Data & 0x00FF 
	fmt.Println("robotID ", robotID, " robot_stand_node: " ,robot_stand_node)
	//call TA SC 
	//TA param
	arg := `{"Args":["TaskAllocation","%d"]}`
    arg = fmt.Sprintf(arg, robotID)

	//call cmd
	cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tacc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
	fmt.Println("cmd done")
	var goal_point Point
	var goal_point_preNodeID int
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
	
	//get payload
	if strings.Contains(str_output, "payload") {
		// 取得payload欄位的值
		payloadStartIndex := strings.Index(str_output, "payload:") + len("payload:")
		payloadStr := str_output[payloadStartIndex:]
		// fmt.Println(payloadStr)
		//trim "" and remove \\
		payloadStr = strings.TrimSpace(payloadStr)
		payloadStr = strings.Trim(payloadStr, "\"")
		payloadStr = strings.ReplaceAll(payloadStr, "\\", "")
		fmt.Println(payloadStr)

		var data map[string]interface{}
		err := json.Unmarshal([]byte(payloadStr), &data)
		if err != nil {
			panic(err)
		}

		goal_point.X = data["x"].(float64)
		goal_point.Y = data["y"].(float64)
		goal_point_preNodeID = int(data["preID"].(float64))

	}
	//have no more task to perform 
	// fmt.Println("goal_point_preNodeID: ", goal_point_preNodeID)
	if goal_point_preNodeID == -87{
		fmt.Println("no task")
		opts := rclgo.NewDefaultPublisherOptions()
		opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
		pub, pubErr := rclNode.NewPublisher("ta_output_waypoint", geo_msgs.PoseArrayTypeSupport, opts)
		if pubErr != nil {
			log.Fatalf("Unable to create publisher: %v", pubErr)
		}
		pub.Publish(geo_msgs.NewPoseArray())
		return
	}
	//how to navi from curPointID to goal_pointID
	var waypoint []Point
	if robot_stand_node != uint32(goal_point_preNodeID){
		//get the path from node curNodeID to frontier perID
		//call TM SC
		arg = `{"Args":["GetShortestPath","%d","%d"]}`
		arg = fmt.Sprintf(arg, robot_stand_node, goal_point_preNodeID)

		//call cmd
		cmd = exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tmcc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)

		
		//out
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error executing command:", err)
			// fmt.Println(string(output))
			return
		}

		str_output = string(output)
		//fetch status number 
		re = regexp.MustCompile(`status:(\d+)`)
		matches = re.FindStringSubmatch(str_output)
		if len(matches) > 1 {
			status = matches[1]
		}

		fmt.Println("got status number: ", status)
		if status != "200"{
			fmt.Println("Chaincode failed ...")
			return 
		}
		
		//get payload
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

			err := json.Unmarshal([]byte(payloadStr), &waypoint)
			if err != nil {
				fmt.Println("unmarsal json waypoint failed")
				return 
			}
			
		}
	}
	
	waypoint = append(waypoint, goal_point)
	fmt.Println("waypoint", waypoint)

	ta_output_waypoint_msg := geo_msgs.NewPoseArray()
	ta_output_waypoint_msg.Header.FrameId = strconv.Itoa(goal_point_preNodeID)
	for _, point := range waypoint{
		pt := geo_msgs.NewPose()
		pt.Position.X = point.X
		pt.Position.Y = point.Y
		ta_output_waypoint_msg.Poses = append(ta_output_waypoint_msg.Poses, *pt)
	}
	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, pubErr := rclNode.NewPublisher("ta_output_waypoint", geo_msgs.PoseArrayTypeSupport, opts)
	if pubErr != nil {
		log.Fatalf("Unable to create publisher: %v", pubErr)
	}
	pub.Publish(ta_output_waypoint_msg)

	fmt.Println("TA done")
}

