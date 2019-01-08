/*
Navicat MySQL Data Transfer

Source Server         : 192.168.195.177
Source Server Version : 50171
Source Host           : 192.168.195.177:3306
Source Database       : rbac

Target Server Type    : MYSQL
Target Server Version : 50171
File Encoding         : 65001

Date: 2018-01-03 15:39:04
*/

SET FOREIGN_KEY_CHECKS=0;

-- ----------------------------
-- Table structure for app
-- ----------------------------
DROP TABLE IF EXISTS `app`;
CREATE TABLE `app` (
  `id` bigint(8) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(32) NOT NULL COMMENT '应用名',
  `user` varchar(32) NOT NULL COMMENT '用户名中文，来自erp',
  `email` varchar(64) NOT NULL COMMENT '创建这个应用的用户邮箱，来自erp',
  `token` varchar(64) NOT NULL COMMENT 'app key',
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT '申请时间',
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT '修改时间',
  `comments` varchar(512) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_name` (`name`) USING HASH,
  KEY `idx_email` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=26 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for resource
-- ----------------------------
DROP TABLE IF EXISTS `resource`;
CREATE TABLE `resource` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint(20) unsigned NOT NULL,
  `name` varchar(32) NOT NULL,
  `comments` varchar(512) NOT NULL,
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_name` (`app_id`,`name`) USING HASH
) ENGINE=InnoDB AUTO_INCREMENT=28 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for role
-- ----------------------------
DROP TABLE IF EXISTS `role`;
CREATE TABLE `role` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint(20) unsigned NOT NULL,
  `name` varchar(32) NOT NULL COMMENT '组名',
  `user_id` bigint(20) unsigned NOT NULL,
  `comments` varchar(64) NOT NULL COMMENT '备注',
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_app_role` (`app_id`,`name`) USING BTREE,
  KEY `idx_app` (`app_id`)
) ENGINE=InnoDB AUTO_INCREMENT=28 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for role_resource
-- ----------------------------
DROP TABLE IF EXISTS `role_resource`;
CREATE TABLE `role_resource` (
  `id` bigint(8) unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint(20) unsigned NOT NULL,
  `role_id` bigint(8) unsigned NOT NULL COMMENT '角色id',
  `resource_id` bigint(8) unsigned NOT NULL COMMENT '资源id',
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_relation` (`app_id`,`role_id`,`resource_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=31 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for role_user
-- ----------------------------
DROP TABLE IF EXISTS `role_user`;
CREATE TABLE `role_user` (
  `id` bigint(8) unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint(20) unsigned NOT NULL,
  `role_id` bigint(8) unsigned NOT NULL COMMENT '角色id',
  `user_id` bigint(8) unsigned NOT NULL COMMENT '资源id',
  `ctime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_relation` (`app_id`,`role_id`,`user_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=40 DEFAULT CHARSET=utf8;

-- ----------------------------
-- Table structure for user
-- ----------------------------
DROP TABLE IF EXISTS `user`;
CREATE TABLE `user` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `app_id` bigint(20) unsigned NOT NULL,
  `name` varchar(32) NOT NULL,
  `email` varchar(64) NOT NULL,
  `ctime` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `mtime` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_email` (`email`,`app_id`) USING BTREE
) ENGINE=InnoDB AUTO_INCREMENT=67 DEFAULT CHARSET=utf8;
