const urlPrefix="/api";

var account;

Date.prototype.Format = function(fmt) 
{
    var o = { 
        "M+" : this.getMonth()+1,                 //月份 
        "d+" : this.getDate(),                    //日 
        "h+" : this.getHours(),                   //小时 
        "m+" : this.getMinutes(),                 //分 
        "s+" : this.getSeconds(),                 //秒 
        "q+" : Math.floor((this.getMonth()+3)/3), //季度 
        "S"  : this.getMilliseconds()             //毫秒 
    }; 
    if(/(y+)/.test(fmt)) 
        fmt=fmt.replace(RegExp.$1, (this.getFullYear()+"").substr(4 - RegExp.$1.length)); 
    for(var k in o) 
        if(new RegExp("("+ k +")").test(fmt)) 
            fmt = fmt.replace(RegExp.$1, (RegExp.$1.length==1) ? (o[k]) : (("00"+ o[k]).substr((""+ o[k]).length))); 
    return fmt; 
}

function bytesToSize(bytes) {
    if (bytes === 0) return '0B';
    k = 1024;
    sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    i = Math.floor(Math.log(bytes) / Math.log(k));

    return (bytes / Math.pow(k, i)).toPrecision(3) + sizes[i];
}

function format(num) {
    return (num+'').replace(/\d{1,3}(?=(\d{3})+(\.\d*)?$)/g, '$&,');
}


function obj2string(o){ 
    var r=[]; 
    if(typeof o=="string"){ 
        return "\""+o.replace(/([\'\"\\])/g,"\\$1").replace(/(\n)/g,"\\n").replace(/(\r)/g,"\\r").replace(/(\t)/g,"\\t")+"\""; 
    } 
    if(typeof o=="object"){ 
        if(!o.sort){ 
            for(var i in o){ 
                r.push(i+":"+obj2string(o[i])); 
            } 
            if(!!document.all&&!/^\n?function\s*toString\(\)\s*\{\n?\s*\[native code\]\n?\s*\}\n?\s*$/.test(o.toString)){ 
                r.push("toString:"+o.toString.toString()); 
            } 
            r="{"+r.join()+"}"; 
        }else{ 
            for(var i=0;i<o.length;i++){ 
                r.push(obj2string(o[i])) 
            } 
            r="["+r.join()+"]"; 
        } 
        return r; 
    } 
    return o.toString(); 
} 


function requestParse(key) { 
    var url = location.href; 
    var paraString = url.substring(url.indexOf("?")+1,url.length).split("&"); 
    var paraObj = {} 
    for (i=0; j=paraString[i]; i++){ 
        paraObj[j.substring(0,j.indexOf("=")).toLowerCase()] = j.substring(j.indexOf("=")+1,j.length); 
    } 
    var returnValue = paraObj[key.toLowerCase()]; 
    if(typeof(returnValue)=="undefined"){ 
        return ""; 
    }else{ 
        if (returnValue.indexOf("#") != -1) {
            return returnValue.substring(0, returnValue.indexOf("#"));
        }
        return returnValue; 
    } 
}


function getIsRequiredName(i) {
    if (i == true) 
        return "必填";
    return "选填";
}

function getPostionName(postion) {
    switch (postion) {
    case 0:
        return "URL";
    case 1:
        return "Header";
    case 2:
    case 3:
        return "Body";
    case 4:
        return "Response JSON";
    case 14:
        return "JSON";
    }
}

//请求方式:0:get, 1:post,2:put,3:delete
function getMethodName(method) {
    switch (method) {
    case 0:
        return "GET";
    case 1:
        return "POST";
    case 2:
        return "PUT";
    case 3:
        return "DELETE";
    case 4:
        return "RESTful";
    }
}

//接口级别:0:重要, 1:普通
function getLevelName(level) {
    switch (level) {
    case 0:
        return "重要";
    case 1:
        return "普通";
    }
}

function getLength(str) {
    return str.replace(/[^ -~]/g, 'AA').length;
}

function limitMaxLength(str, maxLength) {
    var result = [];
    for (var i = 0; i < maxLength; i++) {
        var char = str[i]
            if (/[^ -~]/.test(char))
                maxLength--;
        result.push(char);
    }
    return result.join('');
}

function onInput() {
    if (getLength(this.value) > this.maxlength) {
        this.value = limitMaxLength(this.value, maxLength);
    }
}

//字数统计  
function chEnWordCount(str) {
    //var count = str.replace(/[^\x00-\xff]/g,"**").length;  
    return str.replace(/[^ -~]/g, 'AA').length;
    //return count;  
}  

//英文字母和数字
function checkPath(str) {
    if(!str.match(/^[A-Za-z0-9]{4,40}$/)) {
        return false;
    }
    return true;
}

//字符串非空检查
function checkEmpty(str) {
    if(str.length <= 0 ) {	
        return true;
    }
    return false;
}


