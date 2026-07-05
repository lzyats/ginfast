此目录用于放置 ip2region xdb 数据文件。

默认配置读取：

```yaml
customer:
  ip2region:
    xdb_path: "resource/ip2region/ip2region.xdb"
    v6_xdb_path: ""
```

部署时将 IPv4 数据库文件命名为 `ip2region.xdb` 放入本目录即可启用坐席台客户来源解析。
