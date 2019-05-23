- [1 引言](#1-引言)
	- [1.1	部署监控目的](#11部署监控目的)
- [2 概述](#2-概述)
	- [2.1 什么是Prometheus？](#21-什么是prometheus)
	- [2.2 什么是Grafana？](#22-什么是grafana)
	- [2.3 IRIS Monitor 框架结构如下](#23-iris-monitor-框架结构如下)
- [3 系统部署](#3-系统部署)
	- [3.1 IRIS Hub设置](#31-iris-hub设置)
	- [3.2 安装配置Prometheus](#32-安装配置prometheus)
	- [3.3 安装配置Grafana](#33-安装配置grafana)
- [4 Grafana 使用说明](#4-grafana-使用说明)
	- [4.1 添加阈值Alerting功能](#41-添加阈值alerting功能)





# 1 引言 #
## 1.1	部署监控目的 ##
生产环境中经常会遇到Server负载过高的情况，如果没有监控，很难被立刻发现，直到用户反馈服务器反应速度缓慢，各项数据出了严重问题时才发现，在这种模式下开发较为被动。一套优秀的系统可以在各项指标早期异常的时候就能够发出预警。各方面综合考虑下， 最终选择了IRIS Hub + Prometheus(普罗米修斯)+ Grafana 集成监控方案。

# 2 概述 #
## 2.1 什么是Prometheus？ ##
Prometheus 是开源监控报警系统和时序列数据库. Prometheus在多维度数据收集,记录纯数字时间序列方面表现非常好。它既适用于面向服务器等硬件指标的监控，也适用于高动态的面向服务架构的监控 Prometheus是为服务的可靠性而设计，当服务出现故障时，它可以使你快速定位和诊断问题。另外，值得关注的是它的搭建过程对硬件和服务没有很强的依赖关系。

    Prometheus的特点包括:

-     多维数据模型（时序列数据由metric名和一组key/value组成）
-     目标服务器可以通过发现服务或者静态配置实现
-     通过基于HTTP的(pull)方式采集时序数据
-     多种可视化和仪表盘支持
    
## 2.2 什么是Grafana？ ##
Grafana是在网络架构和应用分析中最流行的时序数据展示工具，它提供了强大和优雅的方式去创建、共享、浏览数据。dashboard中显示了不同metric数据源中的数据。grafana有热插拔控制面板和可扩展的数据源，目前支持绝大部分常用的时序数据库。

## 2.3 IRIS Monitor 框架结构如下 ##
![](https://raw.githubusercontent.com/shilei-wang/monitor/master/image/Monitoring_01.jpg)

# 3 系统部署 #
## 3.1 IRIS Hub设置 ##
    设置配置Monitor参数及端口
    iriscli monitor -n=tcp://localhost:46657
    注意：iris的PRC端口:46657， iriscli暴露给Prometheus端口:26660

	分别打开46657和26660端口:
	ubuntu： 
		ufw allow 46657 && ufw enable
		ufw allow 26660 && ufw enable 
	centOS：
    	firewall-cmd --zone=public --add-port=46657/tcp --permanent
    	firewall-cmd --zone=public --add-port=26660/tcp --permanent	
		firewall-cmd --reload && systemctl restart firewalld.service

可以在浏览器打开http://your ip:26660 来验证，如果能正常读出metrics数据这说明配置成功
    
## 3.2 安装配置Prometheus ##
### 3.2.1 官网下载：  ###
    https://prometheus.io/download/
    https://github.com/prometheus/prometheus 

### 3.2.2 配置prometheus ###
vi prometheus.yml 追加
    
    - job_name: IRIS-Hub
      static_configs:
      - targets: ['your ip:26660']
    	labels:
    	  instance: IRIS-Hub

注意： 
建议设置固定IP，此处不建议填写localhost。

开启服务: 

    cd /安装目录/prometheus
    修改可执行权限：chmod a+x ./prometheus
    运行 ./prometheus （如正常返回："Server is ready to receive web requests."）

打开prometheus 9090 端口:

    ubuntu： 
    	ufw allow 9090 && ufw enable
    centOS：
    	firewall-cmd --zone=public --add-port=9090/tcp --permanent
    	firewall-cmd --reload && systemctl restart firewalld.service

服务开启后，可以先打开浏览器"http://your ip:9090"测试是否配置正确。 下拉框选择“consensus_height”执行execute，如观察到区块高度正常显示，则表示Prometheus安装正确。

## 3.3 安装配置Grafana ##
### 3.3.1 官网下载： ###
    https://grafana.com/grafana/download
	https://github.com/grafana/grafana
	wget https://s3-us-west-2.amazonaws.com/grafana-releases/release/grafana_5.2.1_amd64.deb


注意 ： Grafana需要以下环境才能正常进行开发

    需要先安装libfontconfig 
    ubuntu 
		apt-get install libfontconfig （如有依赖，请执行 apt-get -f install）
	centOS
		yum install fontconfig 

安装Grafana :

    ubuntu：
		dpkg -i grafana_5.2.1_amd64.deb
	centOS:
		rpm -ivh grafana.rpm

### 3.3.2 配置Grafana.ini  ###
vi grafana.ini (Alert配置，以hotmail为例。如暂时没有配置阈值警报服务的需要[email,钉钉等]，此处可先跳过。)

    [smtp]
    enabled = true
    host = smtp-mail.outlook.com:587
    user = ******@hotmail.com
    password = ******
    skip_verify = false
    from_address = ******@hotmail.com
    from_name = Grafana Alert
    
    [alerting]
    enabled = true
    execute_alerts = true
    
    root_url = http://your ip:3000

### 3.3.3 启动 Grafana ###
	
启动:

	service grafana-server start

打开grafana 3000 端口:

    ubuntu： 
    	ufw allow 3000 && ufw enable
    centOS：
    	firewall-cmd --zone=public --add-port=3000/tcp --permanent
    	firewall-cmd --reload && systemctl restart firewalld.service


服务开启后可以先打开浏览器"http://your ip:3000"测试是否配置正确（初始账户 admin 密码 admin）

### 3.3.4添加数据源 ###
点击 "Create your first data source"

    Name: Prometheus
    Type: Prometheus
    URL: http://your ip:9090
    其他：默认

### 3.3.5导入dashboard ###
鼠标滑动到左侧+号，点击Import》upload .json file (json 模板请使用我们提供的最新版本)》下拉框选择Prometheus Datasource 
最后在Home Dashboard面板中点击刚才加载的dashboard，既可以看到最终监控面板。

下载最新dashboard模板:

    https://raw.githubusercontent.com/shilei-wang/monitor/master/IRIS%20HUB%20v1.0.json

![](https://raw.githubusercontent.com/shilei-wang/monitor/master/image/Monitoring_02.jpg)

# 4 Grafana 使用说明 #
## 4.1 添加阈值Alerting功能 ##
### 4.1.1 添加Notification channels ###
点击左侧"铃铛"按钮进入Alerting界面，选择"Notification channels"面板，点击"add channels"添加新的channel。 添加成功之后，channels会以列表形式显示在Notification面板中。 

### 4.1.2 给感兴趣的Metrics Graph添加Alerting Rules ###
首先到DashBoard中点击需要设置预警的Metrics Graph（目前只有图形面板支持警报规则，官网显示在后续版本中会陆续添加到Singlestat和Table面板中）， 点击下拉框中的"Edit", 然后切换到"Alert"面板。

![](https://raw.githubusercontent.com/shilei-wang/monitor/master/image/Monitoring_03.png)

    Name:Alert的标题   
    Evaluate Every:每多少秒监测一次（10s）
    Conditions:  when (max,min,avg) of (query(A<metrics的tag>，2s<时间跨度>，now（距离现在）)
    		     is above  (预警值<如果未到alert rules面板中会显示OK，到了预警会触发notification，并且只会触发一次>)
    test rule : 测试此条规则并返回result log 

为此Alert Rule添加 Notifications：
点击面板中的Notifications按钮，点击send to右侧的"+"号按钮，添加你刚才创建的Channels，这样Alarm的触发消息就会在你添加的channels中通知用户。
    
    特别注意：任何对DashBoard的修改，一定要记得点击面板上端的"Save Dashboard"。
             如需备份也可以点击"share">"export">"save to file"来永久保存此dashboard配置的Json数据。

添加后的rules会以列表形式展现在Alerting中的"Alert Rules"面板下，其中每条rule包含的多种状态，分别是：Alerting、 ok、 pending（待定）、no data、 paused 这5个状态。

值得注意的是如果是ALerting状态，当一次alert被触发后除非手动切换alert rule状态，否则系统不会发出第二次警告。 这里可以手动切换成pending状态。（Grafana目前没有提供自动结束Alerting状态的功能）

### 4.1.3 在Notification channels中添加"Email"通知功能 ###
首先修改ini配置文件中的SMTP设置，修改成功后需要重启服务(参见3.3.2), 进入"Alerting">"Notification channels">"add channels"面板， 填写channel的name， 选择Type为"Email", 勾选"Include Image"，在"Email Address"中填入需要发送的邮箱地址(以;隔开)，点击save保存设置。

另外，可以点击"Send Test"来检验是否设置成功，如果邮件发送成功，会在页面的右上角有发送成功提示。（如发送失败，可以去/var/log/grafana/grafana.log看下错误日志，进行原因排查）

### 4.1.4 在Notification channels中添加"钉钉"通知功能 ###
首先修改ini配置文件中的root_url设置，修改成功后需要重启服务(参见3.3.2)

然后到钉钉中获取WebHook: 钉钉>个人中心>机器人管理>自定义（通过Webhook接入）>添加（填写机器人名字和选择群组）>复制webhook（包含生成的access_token）。

最后进入"add channels"面板，填写channel的name，选择Type为"DingDing", URL处填入刚才获取的复制生成的webhook。 可以使用"Send Test"进行测试， 钉钉中弹出相应的通知说明设置成功。