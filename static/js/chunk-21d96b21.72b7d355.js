(window["webpackJsonp"]=window["webpackJsonp"]||[]).push([["chunk-21d96b21"],{1:function(t,e){},"16dd":function(t,e,i){"use strict";var r=i("38e0"),n=i("9648"),o=i("39d3"),a=i("87ef"),s=i("ad41"),l=[],c=l.sort,u=a((function(){l.sort(void 0)})),d=a((function(){l.sort(null)})),p=s("sort"),f=u||!d||!p;r({target:"Array",proto:!0,forced:f},{sort:function(t){return void 0===t?c.call(o(this)):c.call(o(this),n(t))}})},"2ec0":function(t,e,i){"use strict";i.r(e);var r=function(){var t=this,e=t.$createElement,i=t._self._c||e;return i("div",{staticClass:"app-container"},[i("div",{staticStyle:{"margin-bottom":"10px"}},[i("el-input",{staticClass:"filter-item",staticStyle:{width:"150px"},attrs:{size:"mini",placeholder:t.$t("trade.coin")},nativeOn:{keyup:function(e){return!e.type.indexOf("key")&&t._k(e.keyCode,"enter",13,e.key,"Enter")?null:t.handleFilter(e)}},model:{value:t.listQuery.symbol,callback:function(e){t.$set(t.listQuery,"symbol",e)},expression:"listQuery.symbol"}}),i("el-select",{staticClass:"filter-item",staticStyle:{width:"75px"},attrs:{clearable:"",size:"mini",placeholder:"status"},model:{value:t.listQuery.type,callback:function(e){t.$set(t.listQuery,"type",e)},expression:"listQuery.type"}},[i("el-option",{attrs:{label:t.$t("table.all"),value:"all"}}),i("el-option",{attrs:{label:t.$t("table.open"),value:"open"}}),i("el-option",{attrs:{label:t.$t("table.close"),value:"close"}})],1),i("el-select",{staticClass:"filter-item",staticStyle:{width:"75px"},attrs:{clearable:"",size:"mini",placeholder:"position_side"},model:{value:t.listQuery.position_side,callback:function(e){t.$set(t.listQuery,"position_side",e)},expression:"listQuery.position_side"}},[i("el-option",{attrs:{label:t.$t("table.all"),value:"all"}}),i("el-option",{attrs:{label:t.$t("trade.long"),value:"LONG"}}),i("el-option",{attrs:{label:t.$t("trade.short"),value:"SHORT"}})],1),i("el-date-picker",{staticClass:"filter-item",attrs:{size:"mini",type:"datetime",placeholder:t.$t("table.startTime"),"picker-options":t.pickerOptions},model:{value:t.listQuery.start_time,callback:function(e){t.$set(t.listQuery,"start_time",e)},expression:"listQuery.start_time"}}),i("el-date-picker",{staticClass:"filter-item",attrs:{size:"mini",type:"datetime",placeholder:t.$t("table.endTime"),"picker-options":t.pickerOptions},model:{value:t.listQuery.end_time,callback:function(e){t.$set(t.listQuery,"end_time",e)},expression:"listQuery.end_time"}}),i("el-button",{staticClass:"filter-item",attrs:{size:"mini",type:"primary",icon:"el-icon-search",loading:t.listLoading},on:{click:t.handleFilter}},[t._v(t._s(t.$t("table.search")))]),i("el-button",{attrs:{type:"danger",size:"mini",loading:t.listLoading},on:{click:function(e){return t.deleteAll()}}},[t._v(" "+t._s(t.$t("table.deleteAll"))+" ")]),i("div",{staticStyle:{float:"right","margin-right":"10px"}},[t._v(t._s(t.$t("trade.nowProfit"))+": "+t._s(t.allProfit))])],1),i("el-table",{directives:[{name:"loading",rawName:"v-loading",value:t.listLoading,expression:"listLoading"}],attrs:{data:t.list,"element-loading-text":"Loading",border:"",fit:"",size:"mini","row-key":t.rowKey,"expand-row-keys":t.expandKeys,"highlight-current-row":""},on:{"sort-change":t.sortChange}},[i("el-table-column",{attrs:{label:t.$t("trade.coin"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.symbol)+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.positionSide"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return["LONG"===e.row.position_side?i("span",{staticStyle:{color:"green"}},[t._v(t._s(t.$t("trade."+e.row.position_side.toLowerCase())))]):i("span",{staticStyle:{color:"red"}},[t._v(t._s(t.$t("trade."+e.row.position_side.toLowerCase())))])]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.amount"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.position_amt)+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.usdt"),align:"center",width:"80"},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.usdt)+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.leverage"),align:"center",width:"80"},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.leverage)+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.price"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.price)+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.now_price"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[e.row.close_profit<0?i("span",{staticStyle:{color:"red"}},[t._v(t._s(e.row.now_price))]):i("span",{staticStyle:{color:"green"}},[t._v(t._s(e.row.now_price))])]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.profit"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return["0"==e.row.close_profit?i("span",[t._v("-")]):e.row.close_profit<0?i("span",{staticStyle:{color:"red"}},[t._v(t._s(e.row.close_profit))]):i("span",{staticStyle:{color:"green"}},[t._v(t._s(e.row.close_profit))])]}}])}),i("el-table-column",{attrs:{label:t.$t("position.nowProfit"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[0==e.row.profit_percent?i("span",[t._v("-")]):e.row.profit_percent<0?i("span",{staticStyle:{color:"red"}},[t._v(t._s(e.row.profit_percent))]):i("span",{staticStyle:{color:"green"}},[t._v(t._s(e.row.profit_percent))])]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.close_price"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s("0"!==e.row.close_price?e.row.close_price:"-")+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.period"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.createTime===e.row.updateTime?"-":t.toPeriod(e.row.updateTime,e.row.createTime))+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("trade.time"),align:"center",width:"140","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(t.parseTime(e.row.createTime))+" ")]}}])}),i("el-table-column",{attrs:{label:t.$t("table.actions"),align:"center",width:"80","class-name":"small-padding fixed-width"},scopedSlots:t._u([{key:"default",fn:function(e){var r=e.row;return[i("el-button",{attrs:{type:"danger",size:"mini"},on:{click:function(e){return t.del(r)}}},[t._v(t._s(t.$t("table.delete"))+" ")])]}}])}),i("el-table-column",{attrs:{type:"expand"},scopedSlots:t._u([{key:"default",fn:function(e){return[i("el-form",{staticClass:"info-table-expand",attrs:{"label-position":"left"}},[i("el-form-item",{attrs:{label:""+t.$t("trade.open_strategy")}},[i("codemirror",{staticStyle:{width:"100%"},attrs:{value:e.row.open_strategy,options:t.cmOptions,disabled:""}})],1),e.row.close_strategy?i("el-form-item",{attrs:{label:""+t.$t("trade.close_strategy")}},[i("codemirror",{staticStyle:{width:"100%"},attrs:{value:e.row.close_strategy,options:t.cmOptions}})],1):t._e()],1)]}}])})],1),i("pagination",{directives:[{name:"show",rawName:"v-show",value:t.total>0,expression:"total > 0"}],attrs:{total:t.total,page:t.listQuery.page,limit:t.listQuery.limit},on:{"update:page":function(e){return t.$set(t.listQuery,"page",e)},"update:limit":function(e){return t.$set(t.listQuery,"limit",e)},pagination:t.getList}})],1)},n=[],o=i("8204"),a=i("dd36"),s=i("fee1"),l=(i("e168"),i("4cc3"),i("8b03"),i("16dd"),i("40a4"),i("374d"),i("5227"),i("90c8"),i("fa87"),i("04b0"),i("b775"));function c(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(l["a"])({url:"/test-strategy-results",method:"get",params:t})}function u(){return Object(l["a"])({url:"/test-strategy-results",method:"delete"})}function d(t){return Object(l["a"])({url:"/test-strategy-results/".concat(t),method:"delete"})}var p=i("333d"),f=i("ed08"),m=i("1888"),g=i("1ca5"),_=(i("7392"),i("cedf"),i("d8c5"),i("8f95"),i("31af"),i("e83b"),i("8961"),i("a7ed"),i("cb97"),i("0c7b"),i("065a"),i("5ca6"),i("2399"),i("67ed"),{components:{Pagination:p["a"],codemirror:g["codemirror"]},data:function(){return{cmOptions:{tabSize:2,mode:"text/x-ttcn-cfg",theme:"monokai",lineNumbers:!0,line:!0,foldGutter:!0,lineWrapping:!0,autoFormatJson:!0,jsonIndentation:!0,gutters:["CodeMirror-linenumbers","CodeMirror-foldgutter","CodeMirror-lint-markers"],extraKeys:{Tab:"autocomplete"}},pickerOptions:{shortcuts:[{text:this.$t("table.today"),onClick:function(t){t.$emit("pick",new Date)}},{text:this.$t("table.yesterday"),onClick:function(t){var e=new Date;e.setTime(e.getTime()-864e5),t.$emit("pick",e)}},{text:this.$t("table.lastWeek"),onClick:function(t){var e=new Date;e.setTime(e.getTime()-6048e5),t.$emit("pick",e)}}]},list:[],total:0,listQuery:{page:1,limit:200,sort:"+",start_time:void 0,end_time:void 0,symbol:void 0,type:"all",position_side:"all"},listLoading:!1,rowKey:function(t){return t.symbol+t.id},expandKeys:[]}},computed:{allProfit:function(){var t=this.list.reduce((function(t,e){return t+Number(e.close_profit)}),0);return Object(m["a"])(t,2)}},created:function(){var t=this;return Object(s["a"])(Object(a["a"])().mark((function e(){var i;return Object(a["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return i=localStorage.getItem("test_futures_strategy_search"),i&&(t.listQuery=JSON.parse(i)),e.next=4,t.getList();case 4:case"end":return e.stop()}}),e)})))()},beforeDestroy:function(){},methods:{parseTime:f["c"],toPeriod:function(t,e){var i=(t-e)/1e3/60,r=Math.floor(i/60),n=Math.floor(i%60);return"".concat(this.padTo2Digits(r),":").concat(this.padTo2Digits(n))},padTo2Digits:function(t){return t.toString().padStart(2,"0")},round:function(t){var e=arguments.length>1&&void 0!==arguments[1]?arguments[1]:3;return Object(m["a"])(t,e)},profitPercent:function(t){if("0"===t.close_price){if("LONG"===t.position_side)return(t.now_price-t.price)/t.now_price*t.leverage*100;if("SHORT"===t.position_side)return-(t.now_price-t.price)/t.now_price*t.leverage*100}return"LONG"===t.position_side?(t.close_price-t.price)/t.close_price*t.leverage*100:"SHORT"===t.position_side?-(t.close_price-t.price)/t.close_price*t.leverage*100:void 0},expandChange:function(t,e){this.expandKeys=e.map((function(t){return t.symbol}))},sortChange:function(t){var e=t.order;this.listQuery.sort="ascending"===e?"+":"-",this.getList()},getList:function(){var t=this;return Object(s["a"])(Object(a["a"])().mark((function e(){var i,r,n;return Object(a["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return localStorage.setItem("test_futures_strategy_search",JSON.stringify(t.listQuery)),t.listLoading=!0,e.next=4,c(Object(o["a"])(Object(o["a"])({},t.listQuery),{},{start_time:t.listQuery.start_time?+t.listQuery.start_time:void 0,end_time:t.listQuery.end_time?+t.listQuery.end_time:void 0}));case 4:r=e.sent,n=r.data,t.list=(null!==(i=n.list)&&void 0!==i?i:[]).map((function(e){return e.profit_percent=t.round(t.profitPercent(e)),e.now_profit=t.profit_percent,"0"===e.close_profit&&(e.close_profit=t.round((e.now_price-e.price)*e.position_amt)),e})),t.total=n.total,t.listLoading=!1;case 9:case"end":return e.stop()}}),e)})))()},handleFilter:function(){this.listQuery.page=1,this.getList()},deleteAll:function(){var t=this;this.$confirm("".concat(this.$t("table.confirmDeleteAll"))).then(Object(s["a"])(Object(a["a"])().mark((function e(){return Object(a["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return e.prev=0,e.next=3,u();case 3:return t.$message({message:t.$t("table.actionSuccess"),type:"success"}),e.next=6,t.getList();case 6:e.next=11;break;case 8:e.prev=8,e.t0=e["catch"](0),t.$message({message:t.$t("table.actionFail"),type:"success"});case 11:case"end":return e.stop()}}),e,null,[[0,8]])})))).catch((function(){}))},del:function(t){var e=this;return Object(s["a"])(Object(a["a"])().mark((function i(){return Object(a["a"])().wrap((function(i){while(1)switch(i.prev=i.next){case 0:return i.prev=0,i.next=3,d(t.id);case 3:return e.$message({message:e.$t("table.actionSuccess"),type:"success"}),i.next=6,e.getList();case 6:i.next=11;break;case 8:i.prev=8,i.t0=i["catch"](0),e.$message({message:e.$t("table.actionFail"),type:"success"});case 11:case"end":return i.stop()}}),i,null,[[0,8]])})))()}}}),b=_,y=(i("4925"),i("9bf6")),h=Object(y["a"])(b,r,n,!1,null,"8b253326",null);e["default"]=h.exports},"333d":function(t,e,i){"use strict";var r=function(){var t=this,e=t.$createElement,i=t._self._c||e;return i("div",{staticClass:"pagination-container",class:{hidden:t.hidden}},[i("el-pagination",t._b({attrs:{background:t.background,"current-page":t.currentPage,"page-size":t.pageSize,layout:t.layout,"page-sizes":t.pageSizes,total:t.total},on:{"update:currentPage":function(e){t.currentPage=e},"update:current-page":function(e){t.currentPage=e},"update:pageSize":function(e){t.pageSize=e},"update:page-size":function(e){t.pageSize=e},"size-change":t.handleSizeChange,"current-change":t.handleCurrentChange}},"el-pagination",t.$attrs,!1))],1)},n=[];i("374d");Math.easeInOutQuad=function(t,e,i,r){return t/=r/2,t<1?i/2*t*t+e:(t--,-i/2*(t*(t-2)-1)+e)};var o=function(){return window.requestAnimationFrame||window.webkitRequestAnimationFrame||window.mozRequestAnimationFrame||function(t){window.setTimeout(t,1e3/60)}}();function a(t){document.documentElement.scrollTop=t,document.body.parentNode.scrollTop=t,document.body.scrollTop=t}function s(){return document.documentElement.scrollTop||document.body.parentNode.scrollTop||document.body.scrollTop}function l(t,e,i){var r=s(),n=t-r,l=20,c=0;e="undefined"===typeof e?500:e;var u=function(){c+=l;var t=Math.easeInOutQuad(c,r,n,e);a(t),c<e?o(u):i&&"function"===typeof i&&i()};u()}var c={name:"Pagination",props:{total:{required:!0,type:Number},page:{type:Number,default:1},limit:{type:Number,default:20},pageSizes:{type:Array,default:function(){return[10,20,50,100,200,500]}},layout:{type:String,default:"total, sizes, prev, pager, next, jumper"},background:{type:Boolean,default:!0},autoScroll:{type:Boolean,default:!0},hidden:{type:Boolean,default:!1}},computed:{currentPage:{get:function(){return this.page},set:function(t){this.$emit("update:page",t)}},pageSize:{get:function(){return this.limit},set:function(t){this.$emit("update:limit",t)}}},methods:{handleSizeChange:function(t){this.$emit("pagination",{page:this.currentPage,limit:t}),this.autoScroll&&l(0,800)},handleCurrentChange:function(t){this.$emit("pagination",{page:t,limit:this.pageSize}),this.autoScroll&&l(0,800)}}},u=c,d=(i("c9a5"),i("9bf6")),p=Object(d["a"])(u,r,n,!1,null,"62e794ce",null);e["a"]=p.exports},"40a4":function(t,e,i){var r=i("38e0"),n=i("7b26"),o=i("87ef"),a=n("JSON","stringify"),s=/[\uD800-\uDFFF]/g,l=/^[\uD800-\uDBFF]$/,c=/^[\uDC00-\uDFFF]$/,u=function(t,e,i){var r=i.charAt(e-1),n=i.charAt(e+1);return l.test(t)&&!c.test(n)||c.test(t)&&!l.test(r)?"\\u"+t.charCodeAt(0).toString(16):t},d=o((function(){return'"\\udf06\\ud834"'!==a("\udf06\ud834")||'"\\udead"'!==a("\udead")}));a&&r({target:"JSON",stat:!0,forced:d},{stringify:function(t,e,i){var r=a.apply(null,arguments);return"string"==typeof r?r.replace(s,u):r}})},4925:function(t,e,i){"use strict";i("bcf5")},bcf5:function(t,e,i){},c9a5:function(t,e,i){"use strict";i("dc4d")},dc4d:function(t,e,i){}}]);