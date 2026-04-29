# This SCE registers SAX (Okta) scopes for the GenAIGatewayService 

### NOTE:
<font color="red" size="+2">!!!</font> This SCE must be installed <font color="red" size="+1"> ONLY once </font> on each Control-plane


### <font color="green"> Control Plane GUIDs</font>
GUIDs taken from KH: [How to integrate and use SAX](https://knowledgehub.pega.com/SERVAUTH:How_to_integrate_and_use_SAX)

| Control Plane Stage | Control Plane Account ID | CMDB GUID | Used for                     |
|---------------------|--------------------------|-----------|------------------------------|
| integration         | 444510681578             | <b><font color="red"> 095532be-c9ef-4df1-a1ca-5434c0242944 </font></b> | integration, staging, trials |
| production          | 532396743049             | <b><font color="red"> 00dce86e-66f7-4530-a130-1d9675117258 </font></b> | production, prod-adoption    |
| prod-launchpad      | 987036572573             | <b><font color="red"> 57ec84d0-0852-4c6e-b34e-c96ba2b549df </font></b> | prod-launchpad               |


## SCE Registration procedure for Cloud 2.x/3.x

### 1) Register new Catalog Entry in Provisioning Service 
   - Add SCE using GOC: https://gocinternal.pega.com/prweb/PRAuth/app/CloudKGOC/ssok/case/CCE-20741
     - Group ID = `com.pega.provisioning,services`
     - Catalog Entry Name = `GenAIGatewayServiceSaxRegistration`       
     - Version = `${VERSION}`
    
### 2) Install SCE by adding it to Control-plane (under proper CMDB guid from table)
   - Install SCE using GOC: https://gocinternal.pega.com/prweb/PRAuth/app/CloudKGOC/ssok/service/SV-10704
     - SCE Name = `GenAIGatewayServiceSaxRegistration`
     - Target version = `${VERSION}`
     - Namespace = `default`
