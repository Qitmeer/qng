# Monitoring QNG with InfluxDB and Grafana

This tutorial will help you set up monitoring methods for QNG nodes to better understand their performance and identify potential issues.

### Set INFLUXDB
* Firstly, download and install InfluxDB. [Influxdata download page](https://portal.influxdata.com/downloads/) Multiple download options available.(note:Must be InfluxDB 1.x Open Source)
* After successfully installing InfluxDB, ensure that it runs in the background. By default, it can be accessed through localhost:8086. Before using the Linux client, you must create a new user with administrator privileges. This user will perform advanced management, creating databases and users.

```
influxdb
curl -XPOST "http://localhost:8086/query" --data-urlencode "q=CREATE USER yourname WITH PASSWORD 'yourpassword' WITH ALL PRIVILEGES"
```

* Now, you can use this user's identity to enter the InfluxDB command line through the Linux client。

```
influx -username 'yourname' -password 'yourpassword'
```

* You can directly communicate with InfluxDB through its command line to create databases and users for the QNG indicator

```
create database qng
create user test with password 'test'
```

* Verify the created entries as follows:

```
show databases
show users
```
* Exit the InfluxDB command line.
```
exit
```
* InfluxDB is operating and configured to store metrics from QNG.

### Prepare for QNG

After setting up the database, we need to enable indicator collection in QNG. In this example, we want QNG to push the data to InfluxDB. The basic settings specify the endpoint through which to access InfluxDB and perform database authentication.

```
./qng --metrics --evmenv="--metrics.influxdb"
```

This tag can be attached to the command to start the client or saved to a configuration file.
You can verify whether QNG successfully pushed the data by listing metrics in the database. On the InfluxDB command line:
```
use qng
show measurements
```

### Set GRAFANA
The next step is to install Grafana, which interprets the data through graphics. Follow the installation process in the Grafana documentation for your installation environment. If you do not want to install other versions, make sure to install the OSS version [Download Grafana OSS](https://grafana.com/grafana/download?platform=mac&edition=oss). The following are sample installation steps for installing a distribution version through the resource library:

```
brew update
brew install grafana
brew services restart grafana
```

After Grafana starts running, it should be able to access it at localhost:3000. Use your preferred browser to access this path, and then log in with default credentials (user: admin and password: admin). When prompted, change the default password and save it.
![login](https://ethereum.org/static/40a6aa0c5b23246a5135bc1c4eaac6b0/c1bea/grafana1.png)

You will be redirected to the Grafana homepage. First, set up your source data. Click on the configuration icon in the left column and select 'Data sources'.
![datasource](https://ethereum.org/static/8af6fa7d5e5f84d5638e1a121fcc309d/29114/grafana2.png)

No data source has been created yet. Click on 'Add data source' to define a data source
![adddatasource](https://ethereum.org/static/a0afc059407744fabcdf5c93533d76ee/e3829/grafana3.png)

In this setup, please select 'InfluxDB' and continue with the operation
![select](https://ethereum.org/static/99401599170346a4d47129cc0dac14a1/29114/grafana4.png)

If you run the tool on the same machine, the data source configuration is quite simple. You need to set the InfluxDB address and detailed information to access the database. Please refer to the following figure.
![set](https://ethereum.org/static/b689d089637bf362c9b10ab396dd6132/29114/grafana5.png)

If all operations have been completed and InfluxDB can be accessed, please click on "Save and test" and wait for the confirmation message to pop up.
![save](https://ethereum.org/static/e817953366828b34e02ef4eac08c816e/3a737/grafana6.png)

Now Grafana is set to read data from InfluxDB. At this point, you need to create a dashboard that interprets and displays data. Dashboard attributes are encoded in JSON files, allowing anyone to create and easily import them. On the left column, click on "Import".
* Please use `QNG_Dashboard_grafana.json` in the same directory
![configfile](https://ethereum.org/static/568856acd2ffaf5b4ed0bd16e776cadd/29114/grafana7.png)