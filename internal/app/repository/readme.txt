数据访问层
此层一般由service层调用

此层只用于管理数据。数据从缓存取，还是数据库取，上层service层不关心

gorm模型生成命令如下：
gentool -c "./gen_config"