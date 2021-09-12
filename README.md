# mping
Search the resolved IP addresses of domain names in various places, Ping these IP addresses locally, and find the IP addresses that can successfully establish a connection.
搜索域名在各地的解析IP，并在本地ping这些IP，找出能成功建立连接的IP地址。

## about
这是一个学习golang过程中的练手程序，写法上有可以改进的地方，欢迎朋友们指教。🤝

## run
```code
$ > mping github.com
domain: github.com
需要等待几分钟搜集IP，并分析创建连接相应时间。
If the previous line is garbled, it is not an error, but your system does not support Chinese fonts.
You need to wait a few minutes to collect the IP and analyze the corresponding time to create the connection.
13.229.188.59 min:219.219 max:220.346 avg:219.683
13.250.177.223 min:220.351 max:221.095 avg:220.845
13.114.40.48 min:66.663 max:111.430 avg:89.080
192.30.255.113 min:286.551 max:287.877 avg:287.238
```

