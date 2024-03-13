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
	"strconv"
) 

var cli *channel.Client 
var rclNode *rclgo.Node
type FrontierPoint struct{
	X float64 `json:"x"`
	Y float64 `json:"y"`
	PreNodeID int `json:"preID"`
	DisFromPreNode float64 `json:"distance"`
	Invalid bool `json:"invalid"`
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

	rclNode, rclErr = rclContext.NewNode("frontier_filter", "")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	sub, subErr := rclNode.NewSubscription("frontier_points", geo_msgs.PoseArrayTypeSupport, frontierCallback)
	if subErr != nil {
		log.Fatalf("Unable to create publisher: %v", subErr)
	}


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Spin(ctx, 100)
	defer sub.Close()
	<-doneChannel

}


func frontierCallback(subscription *rclgo.Subscription){
	fmt.Println("frontierCallback")
	msg := geo_msgs.NewPoseArray()
    

    _, err := subscription.TakeMessage(msg)
	if err != nil {
		fmt.Println("Failed to take message:", err)
		return
	}

	if len(msg.Poses) == 0{
		fmt.Println("No frontier point found")
		opts := rclgo.NewDefaultPublisherOptions()
		opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
		pub, pubErr := rclNode.NewPublisher("bc_busy_status", std_msgs.BoolTypeSupport, opts)
		if pubErr != nil {
			log.Fatalf("Unable to create publisher: %v", pubErr)
		}
		msg_status := std_msgs.NewBool()
		msg_status.Data= false
		// fmt.Println("Send bc free status")
		pub.Publish(msg_status)
		return 
	}

	
	fmt.Println("連接這個邊界點的是點編號: ", msg.Header.FrameId)
	preNodeID, err:= strconv.Atoi(msg.Header.FrameId)
	if err != nil {
		fmt.Println("Error during conversion", err)
		return
	}

	//package the frontier points list
	var fPoints []FrontierPoint
	for _, pose := range msg.Poses{
		var fPt FrontierPoint
		fPt.X = pose.Position.X
		fPt.Y = pose.Position.Y
		fPt.PreNodeID = preNodeID
		fPt.DisFromPreNode = pose.Orientation.X
		fPt.Invalid= false
		fPoints = append(fPoints, fPt)
	}

	//transform to JSON format
	pointsJSON, err := json.Marshal(fPoints)
	if err != nil {
		panic(err)
	}
	argsJSON, err := json.Marshal([]interface{}{"Filter", string(pointsJSON)})
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(argsJSON))

	//commandded
	arg := `{"Args":"argsJ"}`
	// 將字串變數 `argsJ` 的內容替換 string(argsJSON)
	arg = strings.Replace(arg, `"argsJ"`, string(argsJSON), 1)
	// fmt.Println(arg)
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

	var filter_outputPoints []FrontierPoint
	if strings.Contains(str_output, "payload") {
		// 取得payload欄位的值
		payloadStartIndex := strings.Index(str_output, "payload:") + len("payload:")
		payloadStr := str_output[payloadStartIndex:]
		fmt.Println(payloadStr)
		//trim "" and remove \\
		payloadStr = strings.TrimSpace(payloadStr)
		payloadStr = strings.Trim(payloadStr, "\"")
		payloadStr = strings.ReplaceAll(payloadStr, "\\", "")
		// fmt.Println(payloadStr)
		// deal with payload 
		err := json.Unmarshal([]byte(payloadStr), &filter_outputPoints)
		if err != nil {
			fmt.Println("unmarsal json waypoint failed")
			return 
		}
	}

	// fmt.Println("filter_outputPoints:  ", filter_outputPoints)



	//call addTask
	if len(filter_outputPoints) > 0{
		//展開json格式
		pointsJSON, err = json.Marshal(filter_outputPoints)
		if err != nil {
			panic(err)
		}

		argsJSON, err = json.Marshal([]interface{}{"AddTask", string(pointsJSON)})
		if err != nil {
			panic(err)
		}

		// fmt.Println(string(argsJSON))

		arg := `{"Args":"argsJ"}`
		// 將字串變數 `argsJ` 的內容替換 string(argsJSON)
		arg = strings.Replace(arg, `"argsJ"`, string(argsJSON), 1)
		// fmt.Println(arg)
		cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tacc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
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
		var status string
		if len(matches) > 1 {
			status = matches[1]
		}

		fmt.Println("got status number: ", status)
		if status != "200"{
			fmt.Println("Chaincode failed ...")
			return 
		}
	}

	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, pubErr := rclNode.NewPublisher("bc_busy_status", std_msgs.BoolTypeSupport, opts)
	if pubErr != nil {
		log.Fatalf("Unable to create publisher: %v", pubErr)
	}
	msg_status := std_msgs.NewBool()
	msg_status.Data= false
	// fmt.Println("Send bc free status")
	pub.Publish(msg_status)
	return 

}

