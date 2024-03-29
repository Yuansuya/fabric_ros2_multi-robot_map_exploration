# fabric_ros2_multi-robot_map_exploration

## Introduction
**該source code為論文Blockchain-Based Multi-Robot Collaborative Map Exploration @NCU by Kevin Zhang (2023)**

該專案採用了區塊鏈作為多機器人資料交換的平台，並採用拓樸地圖(Topological Map)作為多機器人間共享的地圖，達到低傳輸頻寬，並設計了一輕量化任務分配演算法Tiny MinPos，讓多機器人在分配任務時可以有效的避免重複探索。

使用了hyperledger fabric作為區塊鏈平台，ros2作為機器人控制系統。為了將fabric跟ros2串接，採用go application，這讓我可以建立用go語言撰寫的ros2 node，以便調用chaincode的內容。

[機器人自動區域探索source code](https://github.com/Yuansuya/multi-robot_map_exploration/)

Version:
* hyperledger fabric: 2.5
* ros2: foxy
* go: 1.20.2
* Ubuntu: 20.04

## Goal
設計一多機器人協同方法，在探索未知區域時，能達到達成高效探索、高擴展性與高可靠性等成果。

## Demo
採用gazebo作為模擬軟體。

[![demo](https://img.youtube.com/vi/X8ZK3-JHJ0A/0.jpg)](https://www.youtube.com/watch?v=X8ZK3-JHJ0A)   <- youtube link here



## chaincode(CC)

區塊鏈資料交互核心程式，即俗稱的智能合約，寫method/algorithm的地方

### Task(chaincode/Task/task.go)
該CC用來交互關於**未探索點/任務點(task)** 的所有資料，包含了以下prototype
* Set_r_TM_node_filter - 設定過濾的精確度
* AddTask - 新增任務點
* GetTask - 獲取當前所有任務點
* Filter - 用於過濾新的任務點不要再次新增到以探索過的區域
* UpdateTask - 更新任務點
* TaskAllocationGreedy - 貪婪算法的任務分配(用於實驗)
* TaskAllocation - Tiny-MinPos算法的任務分配(本論文提出之算法)

### TopoMap(chaincode/TopoMap/tm.go)
該CC用來交互關於**地圖**的所有資料，包含了以下prototype
* NewRobotJoin - 加入新機器人
* Update - 更新地圖
* GetRobotMapNode - 獲取當前機器人位於地圖之位置
* GetShortestPath - 計算兩點之間的最短路徑之路徑點，輸入為兩個頂點，輸出為一頂點陣列

## invokeCC
每一個資料夾都是一個ros2 node，他會持續執行直到手動shutdown。是區塊鏈(blockchain)跟外界溝通(ros2 nodes)的橋樑，透過ros2溝通的方式(topic publish/subscript)的方式新增/修改資料

* **filter** (invokeCC/filter/filter.go) - 處理任務點過濾
* **getTasks** (invokeCC/getTasks/get.go)- 接收/發佈任務點到topic中
* **initNode** (invokeCC/initNode/init.go)- 初始化機器人狀態，用於當新的機器人加入時將新機器人資訊更新到blockchain
* **taskAllocation** (invokeCC/taskAllocation/ta.go)- 提供任務分配的服務
* **TMget** (invokeCC/TMget/tmtm.go) - 接收/發佈地圖資訊(Topological Map, TM)到topic中
* **updateTM** (invokeCC/updateTM/updateTM.go) - 更新地圖資訊

## node-config
每個機器人節點都需要不同的配置(e.g. ip, key)，透過修改/config/core.yaml調配。
terminalorgX(X= 1, 2, 3)為不同機器人給的環境變數，其中包含不同組織憑證位置, IP, TLS_ENABLE等等

## Contact
Email: kevincolin933@gmail.com
