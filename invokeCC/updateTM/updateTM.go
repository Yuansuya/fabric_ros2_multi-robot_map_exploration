package main

import(
	"fmt"
	"log"
	"sync"
	"context"
    "os/exec"
	"regexp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	geo_msgs "github.com/TIERS/rclgo-msgs/geometry_msgs/msg"
	std_msgs "github.com/TIERS/rclgo-msgs/std_msgs/msg"
	"github.com/TIERS/rclgo/pkg/rclgo"
	"strconv"
	"strings"
) 

var cli *channel.Client 
var rclNode *rclgo.Node
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

	rclNode, rclErr = rclContext.NewNode("newPt", "")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	
	sub, subErr := rclNode.NewSubscription("new_node_at_TM", geo_msgs.PoseArrayTypeSupport, newTMCallback)
	if subErr != nil {
		log.Fatalf("Unable to create publisher: %v", subErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Spin(ctx, 100)
	defer sub.Close()
	<-doneChannel

}


func newTMCallback(subscription *rclgo.Subscription){
	fmt.Printf("TMcallback")
	msg := geo_msgs.NewPoseArray()
    

    _, err := subscription.TakeMessage(msg)

	if err != nil {
		fmt.Println("Failed to take message:", err)
		return
	}
	robotID, err := strconv.Atoi(msg.Header.FrameId)
	if err != nil {
		fmt.Println("Error robotiD during conversion")
		return
	}
    
	fmt.Println("robot: ", robotID, "call update TM")
	for _, cur_add_pose := range(msg.Poses){
		x := cur_add_pose.Position.X
		y := cur_add_pose.Position.Y

		//position x, y and robotID
		arg := `{"Args":["Update","%.6f","%.6f","%d"]}`
		arg = fmt.Sprintf(arg, x, y, robotID)

		//call cmd
		cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tmcc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
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
		var chaincode_status string
		if len(matches) > 1 {
			chaincode_status = matches[1]
		}

		fmt.Println("got status number: ", chaincode_status)
		if chaincode_status != "200"{
			fmt.Println("Chaincode failed ...")
			return 
		}
	}


	//Get now robot located node 
	//call cmd
	arg := `{"Args":["GetRobotMapNode","%d"]}`
	arg = fmt.Sprintf(arg, robotID)
	cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tmcc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
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
	var chaincode_status string
    if len(matches) > 1 {
        chaincode_status = matches[1]
    }

	fmt.Println("got status number: ", chaincode_status)
	if chaincode_status != "200"{
		fmt.Println("Chaincode failed ...")
		return 
	}
	// 取得payload欄位的值
	payloadStartIndex := strings.Index(str_output, "payload:") + len("payload:")
	payloadStr := str_output[payloadStartIndex:]
	// fmt.Println(payloadStr)
	//trim "" and remove \\
	payloadStr = strings.TrimSpace(payloadStr)
	payloadStr = strings.Trim(payloadStr, "\"")
	payloadStr = strings.ReplaceAll(payloadStr, "\\", "")
	// fmt.Println(payloadStr)
	cur_robot_located_node, err := strconv.Atoi(payloadStr)
	if err != nil {
		fmt.Println("Error during conversion")
		return
	}
	fmt.Println("Node robot_cur_located_node ", cur_robot_located_node)


	//update tasks
	//Get now robot located node 
	//call cmd
	arg = `{"Args":["UpdateTask"]}`
	cmd = exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tacc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)
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
        chaincode_status = matches[1]
    }

	fmt.Println("got status number: ", chaincode_status)
	if chaincode_status != "200"{
		fmt.Println("Chaincode failed ...")
		return 
	}

	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	IDpub, IDpubErr := rclNode.NewPublisher("cur_robot_located_node", std_msgs.UInt32TypeSupport, opts)
	if IDpubErr != nil {
		log.Fatalf("Unable to create publisher: %v", IDpubErr)
	}
	IDmsg := std_msgs.NewUInt32()
	IDmsg.Data= uint32(cur_robot_located_node)
	IDpub.Publish(IDmsg)
	return 

}