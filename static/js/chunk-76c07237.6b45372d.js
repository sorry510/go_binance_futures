(window["webpackJsonp"]=window["webpackJsonp"]||[]).push([["chunk-76c07237"],{1:function(e,t){},2465:function(e,t,a){"use strict";a.d(t,"f",(function(){return l})),a.d(t,"k",(function(){return r})),a.d(t,"a",(function(){return o})),a.d(t,"c",(function(){return i})),a.d(t,"d",(function(){return s})),a.d(t,"b",(function(){return c})),a.d(t,"e",(function(){return u})),a.d(t,"j",(function(){return d})),a.d(t,"l",(function(){return b})),a.d(t,"m",(function(){return f})),a.d(t,"g",(function(){return p})),a.d(t,"i",(function(){return m})),a.d(t,"h",(function(){return h}));var n=a("b775");function l(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/features",method:"get",params:e})}function r(e,t){return Object(n["a"])({url:"/features/".concat(e),method:"put",data:t})}function o(e){return Object(n["a"])({url:"/features",method:"post",data:e})}function i(e){return Object(n["a"])({url:"/features/".concat(e),method:"delete"})}function s(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:1;return Object(n["a"])({url:"/features/enable/".concat(e),method:"put"})}function c(e){return Object(n["a"])({url:"/features/batch",method:"put",data:e})}function u(){return Object(n["a"])({url:"/config",method:"get"})}function d(e){return Object(n["a"])({url:"/config",method:"put",data:e})}function b(){return Object(n["a"])({url:"/start",method:"post"})}function f(){return Object(n["a"])({url:"/stop",method:"post"})}function p(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/positions",method:"get",params:e})}function m(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/account",method:"get",params:e})}function h(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/open-orders",method:"get",params:e})}},b5ef:function(e,t,a){"use strict";a.d(t,"a",(function(){return l}));a("1a06"),a("d987"),a("6de1");function n(e,t){if(null==e)return{};var a={};for(var n in e)if({}.hasOwnProperty.call(e,n)){if(t.includes(n))continue;a[n]=e[n]}return a}function l(e,t){if(null==e)return{};var a,l,r=n(e,t);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);for(l=0;l<o.length;l++)a=o[l],t.includes(a)||{}.propertyIsEnumerable.call(e,a)&&(r[a]=e[a])}return r}},e6c4:function(e,t,a){"use strict";a.r(t);var n=function(){var e=this,t=e.$createElement,a=e._self._c||t;return a("div",{staticClass:"app-container"},[a("div",{staticStyle:{"margin-bottom":"10px",display:"flex","justify-content":"space-between","align-items":"center"}},[a("div",{staticStyle:{display:"flex","flex-flow":"row wrap",gap:"10px",width:"75%"}},[a("el-button",{attrs:{type:"success",size:"mini"},on:{click:function(t){return e.openDialog()}}},[e._v(" "+e._s(e.$t("table.add"))+" ")]),a("el-button",{attrs:{type:"success",size:"mini",loading:e.serviceLoading},on:{click:function(t){return e.enableAll(1)}}},[e._v(" "+e._s(e.$t("table.enableAllCoins"))+" ")]),a("el-button",{attrs:{type:"danger",size:"mini",loading:e.serviceLoading},on:{click:function(t){return e.enableAll(0)}}},[e._v(" "+e._s(e.$t("table.disableAllCoins"))+" ")]),a("el-button",{attrs:{type:"success",size:"mini"},on:{click:function(t){e.dialogFormVisible2=!0}}},[e._v(" "+e._s(e.$t("table.editBatch"))+" ")]),a("el-button",{attrs:{type:"success",size:"mini"},on:{click:function(t){return e.$router.push({name:"futuresAccount"})}}},[e._v(" "+e._s(e.$t("route.futuresAccount"))+" ")])],1),a("div",{staticStyle:{width:"25%","text-align":"right"}},[e._v(" "+e._s(e.$t("table.totalCount"))+": "+e._s(e.list.length)+" ")])]),a("div",{staticStyle:{display:"flex","justify-content":"space-between","align-items":"center","margin-bottom":"10px"}},[a("div",{staticStyle:{display:"flex","flex-flow":"row wrap",gap:"10px"}},[a("el-input",{staticClass:"filter-item",staticStyle:{width:"150px"},attrs:{placeholder:e.$t("trade.coin"),size:"small"},model:{value:e.search.symbol,callback:function(t){e.$set(e.search,"symbol",t)},expression:"search.symbol"}}),a("el-select",{staticStyle:{width:"150px"},attrs:{size:"small",placeholder:"status"},model:{value:e.search.enable,callback:function(t){e.$set(e.search,"enable",t)},expression:"search.enable"}},[a("el-option",{attrs:{label:e.$t("table.all"),value:""}}),a("el-option",{attrs:{label:e.$t("table.open"),value:"1"}}),a("el-option",{attrs:{label:e.$t("table.close"),value:"0"}})],1),a("el-select",{staticStyle:{width:"150px"},attrs:{size:"small",placeholder:"margin_type"},model:{value:e.search.margin_type,callback:function(t){e.$set(e.search,"margin_type",t)},expression:"search.margin_type"}},[a("el-option",{attrs:{label:e.$t("table.all"),value:""}}),a("el-option",{attrs:{label:e.$t("trade.ISOLATED"),value:"ISOLATED"}}),a("el-option",{attrs:{label:e.$t("trade.CROSSED"),value:"CROSSED"}})],1),a("el-button",{attrs:{type:"success",size:"mini"},on:{click:e.fetchData}},[e._v(" "+e._s(e.$t("table.search"))+" ")])],1),a("div",{staticStyle:{display:"flex","flex-flow":"row wrap",gap:"10px","align-items":"center"}},[a("el-select",{staticStyle:{width:"80px"},attrs:{size:"small"},on:{change:e.changeRefreshInterval},model:{value:e.interval,callback:function(t){e.interval=t},expression:"interval"}},e._l(30,(function(e){return a("el-option",{key:e,attrs:{label:e+"s",value:e}})})),1),a("span",[e._v(e._s(e.$t("table.refreshInterval")))])],1)]),a("el-table",{directives:[{name:"loading",rawName:"v-loading",value:e.listLoading,expression:"listLoading"}],attrs:{data:e.list,"element-loading-text":"Loading",border:"",fit:"",size:"mini","row-key":e.rowKey,"expand-row-keys":e.expandKeys,"highlight-current-row":""},on:{"sort-change":e.sortChange,"expand-change":e.expandChange}},[a("el-table-column",{attrs:{label:e.$t("trade.coin"),align:"center","show-overflow-tooltip":""},scopedSlots:e._u([{key:"default",fn:function(t){return[e._v(" "+e._s(t.row.symbol)+" ")]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.nowPrice"),align:"center","show-overflow-tooltip":""},scopedSlots:e._u([{key:"default",fn:function(t){return[e._v(" "+e._s(e.round(t.row.close,10))+" ")]}}])}),a("el-table-column",{attrs:{label:"24h↑↓",align:"center","show-overflow-tooltip":"",sortable:"custom"},scopedSlots:e._u([{key:"default",fn:function(t){return[t.row.percentChange<0?a("span",{staticStyle:{color:"red"}},[e._v(e._s(t.row.percentChange)+"%↓ ")]):a("span",{staticStyle:{color:"green"}},[e._v(e._s(t.row.percentChange)+"%↑ ")])]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.klineInterval"),align:"center",width:"110"},scopedSlots:e._u([{key:"default",fn:function(t){return[a("el-select",{attrs:{size:"small"},on:{change:function(a){return e.edit(t.row)}},model:{value:t.row.kline_interval,callback:function(a){e.$set(t.row,"kline_interval",a)},expression:"scope.row.kline_interval"}},[a("el-option",{attrs:{label:"1m",value:"1m"}}),a("el-option",{attrs:{label:"3m",value:"3m"}}),a("el-option",{attrs:{label:"5m",value:"5m"}}),a("el-option",{attrs:{label:"15m",value:"15m"}}),a("el-option",{attrs:{label:"30m",value:"30m"}}),a("el-option",{attrs:{label:"1h",value:"1h"}}),a("el-option",{attrs:{label:"2h",value:"2h"}}),a("el-option",{attrs:{label:"4h",value:"4h"}}),a("el-option",{attrs:{label:"6h",value:"6h"}}),a("el-option",{attrs:{label:"8h",value:"8h"}}),a("el-option",{attrs:{label:"12h",value:"12h"}}),a("el-option",{attrs:{label:"1d",value:"1d"}}),a("el-option",{attrs:{label:"3d",value:"3d"}}),a("el-option",{attrs:{label:"1w",value:"1w"}}),a("el-option",{attrs:{label:"1M",value:"1M"}})],1)]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.marginType"),align:"center",width:"130"},scopedSlots:e._u([{key:"default",fn:function(t){return[a("el-select",{attrs:{size:"small"},on:{change:function(a){return e.edit(t.row)}},model:{value:t.row.marginType,callback:function(a){e.$set(t.row,"marginType",a)},expression:"scope.row.marginType"}},[a("el-option",{attrs:{label:e.$t("trade.ISOLATED"),value:"ISOLATED"}}),a("el-option",{attrs:{label:e.$t("trade.CROSSED"),value:"CROSSED"}})],1)]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.usdt"),align:"center",width:"80"},scopedSlots:e._u([{key:"default",fn:function(t){return[a("el-input",{staticClass:"edit-input",attrs:{size:"small"},on:{blur:function(a){return e.edit(t.row)}},model:{value:t.row.usdt,callback:function(a){e.$set(t.row,"usdt",a)},expression:"scope.row.usdt"}})]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.leverage"),align:"center",width:"80"},scopedSlots:e._u([{key:"default",fn:function(t){return[a("el-input",{staticClass:"edit-input",attrs:{size:"small"},on:{blur:function(a){return e.edit(t.row)}},model:{value:t.row.leverage,callback:function(a){e.$set(t.row,"leverage",a)},expression:"scope.row.leverage"}})]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.profitRate"),align:"center",width:"80"},scopedSlots:e._u([{key:"default",fn:function(t){return[a("el-input",{staticClass:"edit-input",attrs:{size:"small"},on:{blur:function(a){return e.edit(t.row)}},model:{value:t.row.profit,callback:function(a){e.$set(t.row,"profit",a)},expression:"scope.row.profit"}})]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.lossRate"),align:"center",width:"80"},scopedSlots:e._u([{key:"default",fn:function(t){return[a("el-input",{staticClass:"edit-input",attrs:{size:"small"},on:{blur:function(a){return e.edit(t.row)}},model:{value:t.row.loss,callback:function(a){e.$set(t.row,"loss",a)},expression:"scope.row.loss"}})]}}])}),a("el-table-column",{attrs:{label:e.$t("trade.enable"),align:"center",width:"80"},scopedSlots:e._u([{key:"default",fn:function(t){var n=t.row;return[a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6"},on:{change:function(t){return e.isChangeBuy(t,n)}},model:{value:n.enable,callback:function(t){e.$set(n,"enable",t)},expression:"row.enable"}})]}}])}),a("el-table-column",{attrs:{label:e.$t("table.actions"),align:"center",width:"80","class-name":"small-padding fixed-width"},scopedSlots:e._u([{key:"default",fn:function(t){var n=t.row;return[a("el-button",{attrs:{type:"danger",size:"mini"},on:{click:function(t){return e.del(n)}}},[e._v(e._s(e.$t("table.delete"))+" ")])]}}])})],1),a("el-dialog",{attrs:{title:e.dialogTitle,visible:e.dialogFormVisible},on:{"update:visible":function(t){e.dialogFormVisible=t}}},[a("el-form",{ref:"dataForm",staticStyle:{width:"400px","margin-left":"50px"},attrs:{model:e.info,"label-position":"left","label-width":"100px"}},[a("el-form-item",{attrs:{label:e.$t("trade.coin"),prop:"symbol"}},[a("el-input",{model:{value:e.info.symbol,callback:function(t){e.$set(e.info,"symbol",t)},expression:"info.symbol"}})],1)],1),a("div",{staticClass:"dialog-footer",attrs:{slot:"footer"},slot:"footer"},[a("el-button",{on:{click:function(t){e.dialogFormVisible=!1}}},[e._v(e._s(e.$t("table.cancel")))]),a("el-button",{attrs:{type:"primary",loading:e.dialogLoading},on:{click:function(t){return e.addCoin(e.info)}}},[e._v(e._s(e.$t("table.confirm")))])],1)],1),a("el-dialog",{attrs:{title:e.dialogTitle2,visible:e.dialogFormVisible2},on:{"update:visible":function(t){e.dialogFormVisible2=t}}},[a("el-form",{ref:"dataForm2",staticStyle:{width:"400px","margin-left":"50px"},attrs:{model:e.batchInfo,"label-position":"left","label-width":"100px"}},[a("el-form-item",{attrs:{label:e.$t("trade.marginType"),prop:"marginType"}},[a("el-select",{attrs:{size:"small"},model:{value:e.batchInfo.marginType,callback:function(t){e.$set(e.batchInfo,"marginType",t)},expression:"batchInfo.marginType"}},[a("el-option",{attrs:{label:e.$t("trade.ISOLATED"),value:"ISOLATED"}}),a("el-option",{attrs:{label:e.$t("trade.CROSSED"),value:"CROSSED"}})],1)],1),a("el-form-item",{attrs:{label:e.$t("trade.usdt"),prop:"usdt"}},[a("el-input",{model:{value:e.batchInfo.usdt,callback:function(t){e.$set(e.batchInfo,"usdt",t)},expression:"batchInfo.usdt"}})],1),a("el-form-item",{attrs:{label:e.$t("trade.leverage"),prop:"leverage"}},[a("el-input",{model:{value:e.batchInfo.leverage,callback:function(t){e.$set(e.batchInfo,"leverage",t)},expression:"batchInfo.leverage"}})],1),a("el-form-item",{attrs:{label:e.$t("trade.profitRate"),prop:"profit"}},[a("el-input",{model:{value:e.batchInfo.profit,callback:function(t){e.$set(e.batchInfo,"profit",t)},expression:"batchInfo.profit"}})],1),a("el-form-item",{attrs:{label:e.$t("trade.lossRate"),prop:"loss"}},[a("el-input",{model:{value:e.batchInfo.loss,callback:function(t){e.$set(e.batchInfo,"loss",t)},expression:"batchInfo.loss"}})],1)],1),a("div",{staticClass:"dialog-footer",attrs:{slot:"footer"},slot:"footer"},[a("el-button",{on:{click:function(t){e.dialogFormVisible2=!1}}},[e._v(e._s(e.$t("table.cancel")))]),a("el-button",{attrs:{type:"primary",loading:e.dialogLoading2},on:{click:function(t){return e.batchEdit(e.batchInfo)}}},[e._v(e._s(e.$t("table.confirm")))])],1)],1)],1)},l=[],r=a("b5ef"),o=a("8204"),i=a("dd36"),s=a("fee1"),c=(a("e168"),a("4cc3"),a("8b03"),a("16dd"),a("374d"),a("90c8"),a("8d8a"),a("1a71"),a("2465")),u=a("1888"),d=["id","enable","leverage","usdt"],b={data:function(){return{list:[],sort:"+",listLoading:!1,serviceLoading:!1,enableLoading:!1,buyAll:!0,sellAll:!0,search:{symbol:"",enable:"",margin_type:""},dialogFormVisible:!1,dialogLoading:!1,dialogTitle:this.$t("table.add"),info:{},dialogFormVisible2:!1,dialogLoading2:!1,dialogTitle2:this.$t("table.editBatch"),batchInfo:{usdt:void 0,profit:void 0,loss:void 0},rowKey:function(e){return e.symbol},expandKeys:[],timeId:null,interval:30}},computed:{allProfit:function(){var e=this.list.reduce((function(e,t){return e+t.nowProfit}),0);return Object(u["a"])(e,2)}},created:function(){var e=this;return Object(s["a"])(Object(i["a"])().mark((function t(){return Object(i["a"])().wrap((function(t){while(1)switch(t.prev=t.next){case 0:return e.interval=localStorage.getItem("refreshInterval")||30,t.next=3,e.fetchData();case 3:e.timeId=setInterval((function(){return e.fetchData()}),1e3*e.interval);case 4:case"end":return t.stop()}}),t)})))()},beforeDestroy:function(){clearInterval(this.timeId)},methods:{changeRefreshInterval:function(e){var t=this;localStorage.setItem("refreshInterval",e),clearInterval(this.timeId),this.timeId=setInterval((function(){return t.fetchData()}),1e3*e)},round:function(e){var t=arguments.length>1&&void 0!==arguments[1]?arguments[1]:2;return Object(u["a"])(e,t)},expandChange:function(e,t){this.expandKeys=t.map((function(e){return e.symbol}))},sortChange:function(e){var t=e.order;this.sort="ascending"===t?"+":"-",this.fetchData()},fetchData:function(){var e=this;return Object(s["a"])(Object(i["a"])().mark((function t(){var a,n,l;return Object(i["a"])().wrap((function(t){while(1)switch(t.prev=t.next){case 0:return a=e.search,t.next=3,Object(c["f"])(Object(o["a"])({sort:e.sort},a));case 3:n=t.sent,l=n.data,e.list=l.map((function(e){return Object(o["a"])(Object(o["a"])({},e),{},{enable:e.enable>0})}));case 6:case"end":return t.stop()}}),t)})))()},edit:function(e){var t=this;return Object(s["a"])(Object(i["a"])().mark((function a(){var n,l,s,u,b;return Object(i["a"])().wrap((function(a){while(1)switch(a.prev=a.next){case 0:return n=e.id,l=e.enable,s=e.leverage,u=e.usdt,b=Object(r["a"])(e,d),a.prev=1,a.next=4,Object(c["k"])(n,Object(o["a"])(Object(o["a"])({},b),{},{leverage:Number(s),usdt:u,enable:l?1:0}));case 4:return t.$message({message:t.$t("table.editSuccess"),type:"success"}),a.next=7,t.fetchData();case 7:a.next=12;break;case 9:a.prev=9,a.t0=a["catch"](1),t.$message({message:t.$t("table.editFail"),type:"success"});case 12:case"end":return a.stop()}}),a,null,[[1,9]])})))()},del:function(e){var t=this;this.$confirm("".concat(this.$t("table.deleteConfirm")," ").concat(e.symbol,"?")).then(Object(s["a"])(Object(i["a"])().mark((function a(){return Object(i["a"])().wrap((function(a){while(1)switch(a.prev=a.next){case 0:return a.prev=0,a.next=3,Object(c["c"])(e.id);case 3:return t.$message({message:t.$t("table.deleteSuccess"),type:"success"}),a.next=6,t.fetchData();case 6:a.next=11;break;case 8:a.prev=8,a.t0=a["catch"](0),t.$message({message:t.$t("table.deleteFail"),type:"success"});case 11:case"end":return a.stop()}}),a,null,[[0,8]])})))).catch((function(){}))},enableAll:function(e){var t=this,a=1===e?this.$t("table.enableAllCoins"):this.$t("table.disableAllCoins");this.$confirm(a).then(Object(s["a"])(Object(i["a"])().mark((function a(){return Object(i["a"])().wrap((function(a){while(1)switch(a.prev=a.next){case 0:return a.prev=0,a.next=3,Object(c["d"])(e);case 3:return t.$message({message:t.$t("table.actionSuccess"),type:"success"}),a.next=6,t.fetchData();case 6:a.next=11;break;case 8:a.prev=8,a.t0=a["catch"](0),t.$message({message:t.$t("table.actionFail"),type:"success"});case 11:case"end":return a.stop()}}),a,null,[[0,8]])})))).catch((function(){}))},isChangeBuy:function(e,t){var a=this;return Object(s["a"])(Object(i["a"])().mark((function e(){return Object(i["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return e.next=2,a.edit(t);case 2:case"end":return e.stop()}}),e)})))()},openDialog:function(){this.dialogTitle=this.$t("table.add"),this.dialogFormVisible=!0},addCoin:function(e){var t=this;return Object(s["a"])(Object(i["a"])().mark((function a(){var n;return Object(i["a"])().wrap((function(a){while(1)switch(a.prev=a.next){case 0:return n={symbol:e.symbol,quantity:20,percentChange:0,close:0,open:0,low:0,enable:1,updateTime:+new Date},a.next=3,Object(c["a"])(n);case 3:t.dialogFormVisible=!1;case 4:case"end":return a.stop()}}),a)})))()},batchEdit:function(e){var t=this;return Object(s["a"])(Object(i["a"])().mark((function a(){return Object(i["a"])().wrap((function(a){while(1)switch(a.prev=a.next){case 0:return a.prev=0,a.next=3,Object(c["b"])(e);case 3:return t.batchInfo={usdt:void 0,profit:void 0,loss:void 0},t.dialogFormVisible2=!1,a.next=7,t.fetchData();case 7:a.next=12;break;case 9:a.prev=9,a.t0=a["catch"](0),t.$message({message:t.$t("table.actionFail"),type:"success"});case 12:case"end":return a.stop()}}),a,null,[[0,9]])})))()}}},f=b,p=a("9bf6"),m=Object(p["a"])(f,n,l,!1,null,null,null);t["default"]=m.exports}}]);