又拍云本地备份工具
============

##简介
本工具用来将又拍云存储上面的数据备份到本地磁盘。支持增量备份。

##配置
该工具是命令行工具，需要一个配置文件来配合工作。配置文件是JSON格式，内容如下：

```
{
	"user"		:	"<操作员用户名>",
	"password"	:	"<操作员密码>",
	"bucket"	  :	"<空间名称>",
	"localdir"	:	"<本地路径>",
	"domain"	  :	<域名>,
	"routine"	 :	<并发下载数>,
	"debug"	   :	<开启调试模式>
}
```

**注意⚠：上面配置内容中的 `<` 和 `>` 符号表示这里面是参数，实际上不在配置的值中。字符串类型的配置值两边有双引号，整型和布尔型的配置值两边没有双引号。**

| 参数名称 | 类型    | 参数描述                         |备注|
|---------|--------|---------------------------------|---|
| user    | string | 已授权的操作员用户名，从又拍云后台获取| 无|
| password| string | 已授权的操作员密码，从又拍云后台获取  | 无|
| bucket  | string | 空间名称                          | 无|
| localdir| string | 本地存储路径，必须可访问            | 无|
| domain  | int32  | 域名类型，这个值为整数，代表使用哪个又拍云网络|这个根据自己的网络实际情况选择，默认为0，表示自动选择|
| routine | int32  | 并发下载文件的协程数量 | 根据自己的带宽和平均文件大小合理设置，带宽不足设置太大反而慢|
| debug   | bool   | 是否开启调试模式，设置为true时开启，为false时不开启，调试模式可以查看更多详细日志信息|默认为false，不开启|

关于domain的取值，可以参考下面的表：

| 值 |  域名                    | 备注     |
|----|-------------------------|----------|
| 0  | http://v0.api.upyun.com | 自动选择  |
| 1  | http://v1.api.upyun.com | 电信网络  |
| 2  | http://v2.api.upyun.com | 联通网络  |
| 3  | http://v3.api.upyun.com | 移动网络  |

##命令

```
Upyun Backup

Usage:
        Backup the data from upyun bucket

Commands:
        upyun_backup snapshot snapshotFile - 使用又拍云的list接口对又拍云空间里面的文件做一个快照
        upyun_backup backup snapshotFile - 根据生成的快照或者文件路径列表备份文件
```

如果运行命令的时候不加任何参数或者命令错误都会打印上面的命令帮助信息。目前支持的命令有`snapshot`和`backup`。

**备份的方法有两种。**

**第一种方法：**

1. 使用`snapshot`命令并指定一个快照文件来为空间里面的所有文件生成一个快照，快照的内容主要包括文件路径，类型，大小和最后访问时间。
2. 使用`backup`命令并指定一个快照文件开启备份操作。如果中途备份中断，则重新运行命令，对本地已存在的文件将做检测，如果文件没有变化，则跳过，不再重复备份。

第一种方法的快照文件内容（自动生成，不要改动）格式如下：

```
/demo1/jpeg_files2/idcard.jpg	N	290638	1414646065
/demo1/jpeg_files2/多啦爱梦.jpg	N	163794	1414646065
/demo1/jpeg_files2/jemy.jpg	N	120086	1414646064
/demo1/jpeg_files2/jemy.jpeg	N	1957	1414646064
/demo1/jpeg_files2/golang_all.jpg	N	154001	1414646063
```

示例：

```
$ ./upyun_backup snapshot snap1.txt
$ ./upyun_backup backup snap1.txt
```

在运行`snapshot`命令时，如果指定了已经存在的快照文件名，那么会把已经存在的快照文件加上`old`后缀重命名。比如如果指定了快照文件`snap1.txt`，然而这个文件已经存在了，那么已存在的那个文件会被重命名为`snap1.txt.old`，然后再新建一个`snap1.txt`使用。

本来，第一种方式就可以完成备份了，但是这里因为又拍云`list`接口的限制，如果一个目录下面的文件数量超过了1W，那么就无法使用这种方式了。在未来又拍云会更新这个接口，但是现在必须要有办法解决这个问题。这样一来，就有了第二种方法。

**第二种方法：**

在你估算你的目录下面文件超过1W的时候，你就得用这种方法，这种方法其实也很简单，就是你自己从业务数据库里面导出一张所有文件在又拍云的路径表。每个文件的路径格式按照下面的方法提供。

对于这样的链接： `http://bfile.b0.upaiyun.com/jpeg_files/20131011074139e31cf.jpg` ， 你提供的文件路径就是`/jpeg_files/20131011074139e31cf.jpg`，注意这里开头的`/`，结尾不可以有`/`。你从业务数据库导出这些文件的路径，然后放到一个新的文件中，每个文件一行，比如下面这样：

```
/demo1/jpeg_files2/jemy.jpg
/demo1/jpeg_files2/jemy.jpeg
/demo1/jpeg_files2/golang_all.jpg
/demo1/jpeg_files2/golang.jpeg
/demo1/jpeg_files2/golang.jpg
/demo1/jpeg_files2/go.jpg
/demo1/jpeg_files2/ghandi.jpg
/demo1/jpeg_files2/fun.jpg
```
将文件保存一下，比如叫做 `usnap1.txt`，然后还是使用`backup`命令来备份。

```
$ ./upyun_backup backup usnap1.txt
```

**注意⚠：**

无论是上面哪种方式来备份文件，都支持中断备份，就是如果你备份过程中，因为各种原因，备份被迫中断了，那么你下次再次运行备份的时候，只要保证配置文件里面`bucket`和`localdir`的值不变，就会使用增量备份，就是说已备份的那部分数据将不会重复下载备份。





