package main

import(
	"fmt"
	"log"
	"sync"
	"context"
	// "strings"
	"os/exec"
	"regexp"
    // "encoding/json"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	// geo_msgs "github.com/TIERS/rclgo-msgs/geometry_msgs/msg"
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

	rclNode, rclErr = rclContext.NewNode("newRobot", "")
	if rclErr != nil {
		log.Fatal(rclErr)
	}

	sub, subErr := rclNode.NewSubscription("init_robot", std_msgs.Int8TypeSupport, initRobotCallback)
	if subErr != nil {
		log.Fatalf("Unable to create publisher: %v", subErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub.Spin(ctx, 10)
	defer sub.Close()
	<-doneChannel

}


func initRobotCallback(subscription *rclgo.Subscription){
	fmt.Println("initRobotCallback")
	msg := std_msgs.NewInt8()
    

    _, err := subscription.TakeMessage(msg)

	if err != nil {
		fmt.Println("Failed to take message:", err)
		return
	}
	newRobotID := msg.Data
	//call TA SC 
	//TA param
	arg := `{"Args":["NewRobotJoin","%d"]}`
    arg = fmt.Sprintf(arg, newRobotID)

	//call cmd
	cmd := exec.Command("peer", "chaincode", "invoke", "-o", "192.168.0.101:7050", "--ordererTLSHostnameOverride", "orderer1-org0", "--tls", "true", "--cafile", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-C", "mychannel", "-n", "tmcc", "--peerAddresses", "192.168.0.102:7051", "--tlsRootCertFiles", "/media/sf_hyperledger/org1/peer2/tls-msp/tlscacerts/tls-0-0-0-0-7052.pem", "-c", arg)


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


	opts := rclgo.NewDefaultPublisherOptions()
	opts.Qos.Reliability = rclgo.RmwQosReliabilityPolicySystemDefault
	pub, pubErr := rclNode.NewPublisher("bc_busy_status", std_msgs.BoolTypeSupport, opts)
	if pubErr != nil {
		log.Fatalf("Unable to create publisher: %v", pubErr)
	}
	bc_status_msg := std_msgs.NewBool()
	bc_status_msg.Data= false	
	pub.Publish(bc_status_msg)

	fmt.Println("init done")
}

