(window["webpackJsonp"]=window["webpackJsonp"]||[]).push([["chunk-3c45c1ca"],{1:function(t,e){},"16dd":function(t,e,n){"use strict";var o=n("38e0"),r=n("9648"),a=n("39d3"),s=n("87ef"),l=n("ad41"),i=[],u=i.sort,c=s((function(){i.sort(void 0)})),d=s((function(){i.sort(null)})),f=l("sort"),p=c||!d||!f;o({target:"Array",proto:!0,forced:p},{sort:function(t){return void 0===t?u.call(a(this)):u.call(a(this),r(t))}})},"23b0":function(t,e,n){"use strict";n.r(e);var o=function(){var t=this,e=t.$createElement,n=t._self._c||e;return n("div",{staticClass:"app-container"},[n("el-tabs",{model:{value:t.tabName,callback:function(e){t.tabName=e},expression:"tabName"}},[n("el-tab-pane",{attrs:{label:t.$t("trade.assets"),name:"account"}},[n("el-table",{attrs:{data:t.account.assets,"element-loading-text":"Loading",border:"",fit:"",size:"mini","row-key":t.rowKey,"highlight-current-row":""}},[n("el-table-column",{attrs:{label:t.$t("assets.asset"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.asset)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.walletBalance"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.walletBalance)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.unrealizedProfit"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.unrealizedProfit)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.marginBalance"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.marginBalance)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.availableBalance"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.availableBalance)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.maxWithdrawAmount"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.maxWithdrawAmount)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.updateTime"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(t.parseTime(e.row.updateTime))+" ")]}}])})],1)],1),n("el-tab-pane",{attrs:{label:t.$t("trade.position"),name:"position"}},[n("div",{staticStyle:{display:"flex","justify-content":"space-between","align-items":"center","margin-bottom":"10px"}},[n("div",{staticStyle:{display:"flex","flex-flow":"row wrap",gap:"10px","justify-content":"center","align-items":"center"}},[n("el-input",{staticClass:"filter-item",staticStyle:{width:"150px"},attrs:{placeholder:t.$t("trade.coin"),size:"small"},model:{value:t.search.symbol,callback:function(e){t.$set(t.search,"symbol",e)},expression:"search.symbol"}}),n("el-button",{attrs:{type:"success",size:"mini"},on:{click:t.searchFuturesPositions}},[t._v(" "+t._s(t.$t("table.search"))+" ")]),n("div",{staticStyle:{float:"right","margin-right":"20px"}},[t._v(t._s(t.$t("trade.nowProfit"))+": "+t._s(t.allProfit))])],1),n("div",{staticStyle:{display:"flex","flex-flow":"row wrap",gap:"10px","align-items":"center"}},[n("el-select",{staticStyle:{width:"80px"},attrs:{size:"small"},on:{change:t.changeRefreshInterval},model:{value:t.interval,callback:function(e){t.interval=e},expression:"interval"}},t._l(30,(function(t){return n("el-option",{key:t,attrs:{label:t+"s",value:t}})})),1),n("span",[t._v(t._s(t.$t("table.refreshInterval")))])],1)]),n("el-table",{attrs:{data:t.positions,"element-loading-text":"Loading",border:"",fit:"",size:"mini","row-key":t.rowKey,"expand-row-keys":t.expandKeys,"highlight-current-row":""},on:{"sort-change":t.sortChange,"expand-change":t.expandChange}},[n("el-table-column",{attrs:{label:t.$t("position.symbol"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.symbol)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.positionAmt"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.positionAmt)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.entryPrice"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.entryPrice)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.markPrice"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.markPrice)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.liquidationPrice"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.liquidationPrice)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.unrealizedProfit"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[e.row.nowProfit<0?n("span",{staticStyle:{color:"red"}},[t._v(t._s(e.row.unRealizedProfit))]):n("span",{staticStyle:{color:"green"}},[t._v(t._s(e.row.unRealizedProfit))])]}}])}),n("el-table-column",{attrs:{label:t.$t("position.nowProfit"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[e.row.nowProfit<0?n("span",{staticStyle:{color:"red"}},[t._v(t._s(e.row.nowProfit))]):n("span",{staticStyle:{color:"green"}},[t._v(t._s(e.row.nowProfit))])]}}])}),n("el-table-column",{attrs:{label:t.$t("position.leverage"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.leverage)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.positionSide"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(t.$t("positionSide."+e.row.positionSide))+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("position.isolated"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(0!=e.row.isolatedWallet?t.$t("position.isolated"):t.$t("position.crossed"))+" ")]}}])})],1)],1),n("el-tab-pane",{attrs:{label:t.$t("trade.openOrder"),name:"openOrder"}},[n("el-table",{attrs:{data:t.openOrders,"element-loading-text":"Loading",border:"",fit:"",size:"mini","row-key":t.rowKey,"highlight-current-row":""}},[n("el-table-column",{attrs:{label:t.$t("order.symbol"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.symbol)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("order.price"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.price)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("order.origQty"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.origQty)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("order.executedQty"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(e.row.executedQty)+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("order.side"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(t.$t("side."+e.row.side))+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("order.positionSide"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(t.$t("positionSide."+e.row.positionSide))+" ")]}}])}),n("el-table-column",{attrs:{label:t.$t("assets.updateTime"),align:"center","show-overflow-tooltip":""},scopedSlots:t._u([{key:"default",fn:function(e){return[t._v(" "+t._s(t.parseTime(e.row.updateTime))+" ")]}}])})],1)],1)],1)],1)},r=[],a=n("8204"),s=n("dd36"),l=n("fee1"),i=(n("3dd5"),n("d987"),n("4cc3"),n("8b03"),n("16dd"),n("374d"),n("90c8"),n("8d8a"),n("6de1"),n("1a71"),n("2465")),u=n("ed08"),c=n("1888"),d={data:function(){return{tabName:"position",account:{assets:[],positions:[]},positions:[],openOrders:[],search:{symbol:""},sort:"+",rowKey:function(t){return t.positionSide?t.symbol+t.positionSide:t.symbol},expandKeys:[],timeId:null,interval:30}},computed:{allProfit:function(){var t=this.positions.reduce((function(t,e){return t+Number(e.unrealized_profit)}),0);return Object(c["a"])(t,2)}},watch:{tabName:function(t){this.fetchData(t)}},created:function(){var t=this;return Object(l["a"])(Object(s["a"])().mark((function e(){return Object(s["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return t.interval=localStorage.getItem("accountRefreshInterval")||30,e.next=3,t.fetchData();case 3:case"end":return e.stop()}}),e)})))()},beforeDestroy:function(){clearInterval(this.timeId)},methods:{parseTime:u["c"],changeRefreshInterval:function(t){localStorage.setItem("accountRefreshInterval",t),this.fetchData()},round:function(t){var e=arguments.length>1&&void 0!==arguments[1]?arguments[1]:2;return Object(c["a"])(t,e)},expandChange:function(t,e){this.expandKeys=e.map((function(t){return t.symbol}))},sortChange:function(t){var e=t.order;this.sort="ascending"===e?"+":"-",this.fetchData()},fetchData:function(){var t=arguments,e=this;return Object(l["a"])(Object(s["a"])().mark((function n(){var o;return Object(s["a"])().wrap((function(n){while(1)switch(n.prev=n.next){case 0:o=t.length>0&&void 0!==t[0]?t[0]:null,e.timeId&&clearInterval(e.timeId),o||(o=e.tabName),"account"===o?e.getFuturesAccount():"position"===o?e.getFuturesPositions():"openOrder"===o&&e.getFuturesOpenOrders(),e.timeId=setInterval((function(){"account"===o?e.getFuturesAccount():"position"===o?e.getFuturesPositions():"openOrder"===o&&e.getFuturesOpenOrders()}),1e3*e.interval);case 5:case"end":return n.stop()}}),n)})))()},getFuturesAccount:function(){var t=this;return Object(l["a"])(Object(s["a"])().mark((function e(){var n,o,r,a,l,u;return Object(s["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return e.next=2,Object(i["h"])();case 2:n=e.sent,o=n.data,r=o.assets,a=void 0===r?[]:r,l=o.positions,u=void 0===l?[]:l,t.account={assets:a,positions:u};case 9:case"end":return e.stop()}}),e)})))()},searchFuturesPositions:function(){var t=this;return Object(l["a"])(Object(s["a"])().mark((function e(){var n;return Object(s["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:n=t.positions.filter((function(e){var n=!0;return t.search.symbol&&(console.log(e.symbol,t.search.symbol),n=n&&!e.symbol.includes(t.search.symbol)),n})),t.positions=n;case 2:case"end":return e.stop()}}),e)})))()},getFuturesPositions:function(){var t=this;return Object(l["a"])(Object(s["a"])().mark((function e(){var n,o,r;return Object(s["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return e.next=2,Object(i["j"])();case 2:n=e.sent,o=n.data.positions,r=void 0===o?[]:o,t.positions=r.map((function(e){var n=Math.abs(e.positionAmt),o=Number(e.unRealizedProfit),r=Number(e.leverage),s=Number(e.markPrice),l=o/(n*s)*r*100;return Object(a["a"])(Object(a["a"])({},e),{},{unRealizedProfit:t.round(o,2),nowProfit:t.round(l,6)})})),t.search={symbol:""};case 7:case"end":return e.stop()}}),e)})))()},getFuturesOpenOrders:function(){var t=this;return Object(l["a"])(Object(s["a"])().mark((function e(){var n,o;return Object(s["a"])().wrap((function(e){while(1)switch(e.prev=e.next){case 0:return e.next=2,Object(i["i"])();case 2:n=e.sent,o=n.data.openOrders,t.openOrders=o||[];case 5:case"end":return e.stop()}}),e)})))()}}},f=d,p=n("9bf6"),b=Object(p["a"])(f,o,r,!1,null,null,null);e["default"]=b.exports},2465:function(t,e,n){"use strict";n.d(e,"g",(function(){return r})),n.d(e,"n",(function(){return a})),n.d(e,"a",(function(){return s})),n.d(e,"c",(function(){return l})),n.d(e,"e",(function(){return i})),n.d(e,"b",(function(){return u})),n.d(e,"f",(function(){return c})),n.d(e,"m",(function(){return d})),n.d(e,"o",(function(){return f})),n.d(e,"p",(function(){return p})),n.d(e,"h",(function(){return b})),n.d(e,"j",(function(){return m})),n.d(e,"i",(function(){return h})),n.d(e,"l",(function(){return w})),n.d(e,"r",(function(){return v})),n.d(e,"d",(function(){return g})),n.d(e,"k",(function(){return _})),n.d(e,"q",(function(){return y}));var o=n("b775");function r(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(o["a"])({url:"/features",method:"get",params:t})}function a(t,e){return Object(o["a"])({url:"/features/".concat(t),method:"put",data:e})}function s(t){return Object(o["a"])({url:"/features",method:"post",data:t})}function l(t){return Object(o["a"])({url:"/features/".concat(t),method:"delete"})}function i(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:1;return Object(o["a"])({url:"/features/enable/".concat(t),method:"put"})}function u(t){return Object(o["a"])({url:"/features/batch",method:"put",data:t})}function c(){return Object(o["a"])({url:"/config",method:"get"})}function d(t){return Object(o["a"])({url:"/config",method:"put",data:t})}function f(){return Object(o["a"])({url:"/start",method:"post"})}function p(){return Object(o["a"])({url:"/stop",method:"post"})}function b(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(o["a"])({url:"/futures/account",method:"get",params:t})}function m(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(o["a"])({url:"/futures/positions",method:"get",params:t})}function h(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(o["a"])({url:"/futures/open-orders",method:"get",params:t})}function w(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(o["a"])({url:"/futures/local/positions",method:"get",params:t})}function v(t,e){return Object(o["a"])({url:"/futures/local/positions/".concat(t),method:"put",data:e})}function g(t,e){return Object(o["a"])({url:"/futures/local/positions/".concat(t),method:"delete",data:e})}function _(){var t=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(o["a"])({url:"/futures/local/open-orders",method:"get",params:t})}function y(t,e){return Object(o["a"])({url:"/features/strategy-rule/test/".concat(t),method:"post",data:e})}}}]);