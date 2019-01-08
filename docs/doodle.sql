
SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for admin
-- ----------------------------
DROP TABLE IF EXISTS `admin`;
CREATE TABLE `admin` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user` varchar(32) NOT NULL COMMENT '用户名',
  `email` varchar(64) NOT NULL COMMENT '用户邮箱',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for application
-- ----------------------------
DROP TABLE IF EXISTS `application`;
CREATE TABLE `application` (
  `id` bigint(8) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) NOT NULL COMMENT '应用名',
  `user` varchar(32) NOT NULL COMMENT '用户名中文，来自erp',
  `email` varchar(64) NOT NULL COMMENT '创建这个应用的用户邮箱，来自erp',
  `token` varchar(64) NOT NULL DEFAULT ' ' COMMENT 'app key',
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `comments` varchar(512) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_name` (`name`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for deploy
-- ----------------------------
DROP TABLE IF EXISTS `deploy`;
CREATE TABLE `deploy` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint(20) unsigned NOT NULL DEFAULT '0',
  `server` varchar(64) NOT NULL DEFAULT '',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=MyISAM AUTO_INCREMENT=4 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for distributor
-- ----------------------------
DROP TABLE IF EXISTS `distributor`;
CREATE TABLE `distributor` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint(20) NOT NULL,
  `state` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '0.未开始\r\n1.开始编译\r\n2.编译成功\r\n3.编译出错\r\n4.开始安装\r\n5.安装成功\r\n6.安装出错\r\n',
  `server` varchar(64) NOT NULL,
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=MyISAM AUTO_INCREMENT=23 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for distributor_logs
-- ----------------------------
DROP TABLE IF EXISTS `distributor_logs`;
CREATE TABLE `distributor_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `distributor_id` bigint(20) unsigned NOT NULL,
  `state` int(10) unsigned NOT NULL DEFAULT '0',
  `pid` int(10) unsigned NOT NULL,
  `info` text NOT NULL,
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_distributor_id` (`distributor_id`) USING BTREE
) ENGINE=MyISAM AUTO_INCREMENT=102 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for interface
-- ----------------------------
DROP TABLE IF EXISTS `interface`;
CREATE TABLE `interface` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `project_id` bigint(20) unsigned NOT NULL,
  `name` varchar(32) NOT NULL COMMENT '接口名称',
  `user` varchar(32) NOT NULL DEFAULT '',
  `email` varchar(64) NOT NULL DEFAULT '',
  `state` tinyint(1) unsigned NOT NULL DEFAULT '0' COMMENT '状态0:未发布，1：发布,2:后端异常',
  `version` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '0:原接口平台转发类接口\r\n1:faas类自注册接口',
  `method` tinyint(1) unsigned NOT NULL COMMENT '请求方式:0:get, 1:post,2:put,3:delete',
  `path` varchar(64) NOT NULL COMMENT '接口路径',
  `backend` varchar(64) NOT NULL COMMENT '实际接口地址',
  `comments` varchar(512) NOT NULL DEFAULT '',
  `level` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0:重要,1:普通',
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_path` (`project_id`,`path`,`method`) USING BTREE,
  KEY `idx_project_id` (`project_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=45 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for module
-- ----------------------------
DROP TABLE IF EXISTS `module`;
CREATE TABLE `module` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `url` varchar(128) NOT NULL DEFAULT '',
  `project_id` bigint(20) unsigned NOT NULL,
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '编译的应用名',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_project_id` (`url`) USING BTREE
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for project
-- ----------------------------
DROP TABLE IF EXISTS `project`;
CREATE TABLE `project` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `user` varchar(32) NOT NULL COMMENT '管理员信息， 中文',
  `email` varchar(64) NOT NULL COMMENT '项目管理者邮箱',
  `version` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '0:原始接口平台接口\r\n1:新faas接口',
  `source` varchar(128) NOT NULL DEFAULT '',
  `path` varchar(32) NOT NULL DEFAULT '',
  `comments` varchar(512) NOT NULL DEFAULT '',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `role_id` bigint(20) unsigned NOT NULL,
  `resource_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_path` (`path`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for relation
-- ----------------------------
DROP TABLE IF EXISTS `relation`;
CREATE TABLE `relation` (
  `id` bigint(8) unsigned NOT NULL AUTO_INCREMENT,
  `interface_id` bigint(8) unsigned NOT NULL,
  `application_id` bigint(8) unsigned NOT NULL,
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_relation` (`interface_id`,`application_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for variable
-- ----------------------------
DROP TABLE IF EXISTS `variable`;
CREATE TABLE `variable` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `interface_id` bigint(20) NOT NULL COMMENT '接口id',
  `postion` tinyint(1) unsigned NOT NULL COMMENT '0:url参数\r\n1:header参数2:post body',
  `name` varchar(64) NOT NULL COMMENT '字段名',
  `is_number` tinyint(1) NOT NULL COMMENT '0:string, 1:number',
  `is_required` tinyint(1) NOT NULL COMMENT '0:可选，1：必选',
  `example` varchar(64) NOT NULL COMMENT '示例',
  `comments` varchar(512) DEFAULT NULL,
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `mtime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_interface_id` (`interface_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
