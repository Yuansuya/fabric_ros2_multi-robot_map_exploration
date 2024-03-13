# fabric_ros2_multi-robot_map_exploration

## Introduction
**該source code為論文Blockchain-Based Multi-Robot Collaborative Map Exploration @NCU by Kevin Zhang (2023)**

使用了hyperledger fabric作為區塊鏈平台，ros2作為機器人控制系統。為了將fabric跟ros2串接，採用go application，這讓我可以建立用go語言撰寫的ros2 node，以便調用chaincode的內容。

Version:
* hyperledger fabric: 2.5
* go: 1.20.2
* ros2: foxy


## chaincode(CC)

區塊鏈交互核心程式，即俗稱的智能合約

### Task
該CC用來交互關於==未探索點/任務點(task)==的所有資料，包含了以下prototype
* Set_r_TM_node_filter - 設定過濾的精確度
* AddTask - 新增任務點
* GetTask - 獲取當前所有任務點
* Filter - 用於過濾新的任務點不要再次新增到以探索過的區域
* UpdateTask - 更新任務點
* TaskAllocationGreedy - 貪婪算法的任務分配(用於實驗)
* TaskAllocation - Tiny-MinPos算法的任務分配(本論文提出之算法)

### TopoMap
該CC用來交互關於==地圖==的所有資料，包含了以下prototype
* NewRobotJoin - 加入新機器人
* Update - 更新地圖
* GetRobotMapNode - 獲取當前機器人位於地圖之位置
* GetShortestPath - 計算兩點之間的最短路徑之路徑點，輸入為兩個頂點，輸出為一頂點陣列

