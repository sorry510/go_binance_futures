(window["webpackJsonp"]=window["webpackJsonp"]||[]).push([["chunk-37fda406"],{2465:function(e,t,a){"use strict";a.d(t,"f",(function(){return i})),a.d(t,"m",(function(){return s})),a.d(t,"a",(function(){return o})),a.d(t,"c",(function(){return c})),a.d(t,"d",(function(){return r})),a.d(t,"b",(function(){return l})),a.d(t,"e",(function(){return u})),a.d(t,"l",(function(){return d})),a.d(t,"n",(function(){return f})),a.d(t,"o",(function(){return v})),a.d(t,"g",(function(){return g})),a.d(t,"i",(function(){return p})),a.d(t,"h",(function(){return b})),a.d(t,"k",(function(){return h})),a.d(t,"j",(function(){return m})),a.d(t,"p",(function(){return _}));var n=a("b775");function i(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/features",method:"get",params:e})}function s(e,t){return Object(n["a"])({url:"/features/".concat(e),method:"put",data:t})}function o(e){return Object(n["a"])({url:"/features",method:"post",data:e})}function c(e){return Object(n["a"])({url:"/features/".concat(e),method:"delete"})}function r(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:1;return Object(n["a"])({url:"/features/enable/".concat(e),method:"put"})}function l(e){return Object(n["a"])({url:"/features/batch",method:"put",data:e})}function u(){return Object(n["a"])({url:"/config",method:"get"})}function d(e){return Object(n["a"])({url:"/config",method:"put",data:e})}function f(){return Object(n["a"])({url:"/start",method:"post"})}function v(){return Object(n["a"])({url:"/stop",method:"post"})}function g(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/account",method:"get",params:e})}function p(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/positions",method:"get",params:e})}function b(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/open-orders",method:"get",params:e})}function h(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/local/positions",method:"get",params:e})}function m(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(n["a"])({url:"/futures/local/open-orders",method:"get",params:e})}function _(e,t){return Object(n["a"])({url:"/features/strategy-rule/test/".concat(e),method:"post",data:t})}},"5fe0":function(e,t,a){"use strict";a("6aac")},"6aac":function(e,t,a){},fb50:function(e,t,a){"use strict";a.r(t);var n=function(){var e=this,t=e.$createElement,a=e._self._c||t;return a("div",{staticClass:"dashboard-container"},[a("el-collapse",{directives:[{name:"loading",rawName:"v-loading",value:e.loading,expression:"loading"}],attrs:{value:["futures","spot","delivery","new_coin_rush","debug","external"]}},[a("el-collapse-item",{attrs:{name:"futures"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.futuresTrade"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"future_enable")}},model:{value:e.config.tradeFutureEnable,callback:function(t){e.$set(e.config,"tradeFutureEnable",t)},expression:"config.tradeFutureEnable"}})],1)]),a("div",{staticClass:"dashboard-text"},[a("div",{staticStyle:{"margin-left":"20px"}},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.websocket"))+": ")]),a("el-switch",{attrs:{value:e.config.wsFuturesEnable,"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"ws_futures_enable")}}}),a("span",{staticClass:"green",staticStyle:{"margin-left":"20px"}},[e._v(e._s(e.$t("showPage.autoUpdatePrice")))])],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.futuresPositionConvertEnable"))+": ")]),a("el-switch",{attrs:{value:e.config.futuresPositionConvertEnable,"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"futures_position_convert_enable")}}})],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.allowLong"))+": ")]),a("el-switch",{attrs:{value:e.config.coinAllowLong,"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"future_allow_long")}}})],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.allowShort"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"future_allow_short")}},model:{value:e.config.coinAllowShort,callback:function(t){e.$set(e.config,"coinAllowShort",t)},expression:"config.coinAllowShort"}})],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.strategyTrade"))+": ")]),a("el-select",{staticStyle:{width:"100px"},attrs:{size:"small"},on:{change:function(t){return e.editConfig(t,"future_strategy_trade")}},model:{value:e.config.tradeStrategyTrade,callback:function(t){e.$set(e.config,"tradeStrategyTrade",t)},expression:"config.tradeStrategyTrade"}},[a("el-option",{attrs:{label:e.$t("strategyType.line1"),value:"line1"}}),a("el-option",{attrs:{label:e.$t("strategyType.line2"),value:"line2"}}),a("el-option",{attrs:{label:e.$t("strategyType.line3"),value:"line3"}}),a("el-option",{attrs:{label:e.$t("strategyType.line4"),value:"line4"}}),a("el-option",{attrs:{label:e.$t("strategyType.line5"),value:"line5"}}),a("el-option",{attrs:{label:e.$t("strategyType.line6"),value:"line6"}}),a("el-option",{attrs:{label:e.$t("strategyType.line7"),value:"line7"}})],1)],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.strategyCoin"))+": ")]),a("el-select",{staticStyle:{width:"100px"},attrs:{size:"small"},on:{change:function(t){return e.editConfig(t,"future_strategy_coin")}},model:{value:e.config.tradeStrategyCoin,callback:function(t){e.$set(e.config,"tradeStrategyCoin",t)},expression:"config.tradeStrategyCoin"}},[a("el-option",{attrs:{label:e.$t("coin1"),value:"coin1"}}),a("el-option",{attrs:{label:e.$t("coin2"),value:"coin2"}}),a("el-option",{attrs:{label:e.$t("coin3"),value:"coin3"}}),a("el-option",{attrs:{label:e.$t("coin4"),value:"coin4"}}),a("el-option",{attrs:{label:e.$t("coin5"),value:"coin5"}}),a("el-option",{attrs:{label:e.$t("coin6"),value:"coin6"}})],1)],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.positionMaxCount"))+": ")]),a("el-input",{staticStyle:{width:"75px"},attrs:{type:"number"},on:{change:function(t){return e.editConfig(t,"future_max_count")}},model:{value:e.config.coinMaxCount,callback:function(t){e.$set(e.config,"coinMaxCount",t)},expression:"config.coinMaxCount"}})],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.excludeSymbols"))+": ")]),a("el-select",{staticStyle:{width:"80%"},attrs:{multiple:"",filterable:"",size:"small"},on:{change:function(t){return e.editConfig(t,"future_exclude_symbols")}},model:{value:e.excludeSymbols,callback:function(t){e.excludeSymbols=t},expression:"excludeSymbols"}},e._l(e.symbols,(function(e){return a("el-option",{key:e,attrs:{label:e,value:e}})})),1)],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.orderType"))+": ")]),a("el-select",{staticStyle:{width:"100px"},attrs:{size:"small"},on:{change:function(t){return e.editConfig(t,"future_order_type")}},model:{value:e.config.coinOrderType,callback:function(t){e.$set(e.config,"coinOrderType",t)},expression:"config.coinOrderType"}},[a("el-option",{attrs:{label:e.$t("MARKET"),value:"MARKET"}}),a("el-option",{attrs:{label:e.$t("LIMIT"),value:"LIMIT"}})],1),a("span",{staticClass:"green",staticStyle:{"margin-left":"20px"}},[e._v(e._s("LIMIT"===e.config.coinOrderType?e.$t("showPage.limitOrderType"):e.$t("showPage.marketOrderType")))])],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.testStrategy"))+": ")]),a("el-switch",{attrs:{value:e.config.tradeFutureTest,"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"future_test")}}}),e.config.tradeFutureTest?a("el-button",{staticStyle:{"margin-left":"10px"},attrs:{type:"success",size:"mini"},on:{click:function(t){return e.$router.push({name:"testStrategyResult"})}}},[e._v(" "+e._s(e.$t("route.testStrategyResult"))+" ")]):e._e()],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.testStrategyNoticeLimitMin"))+": ")]),a("el-input",{staticStyle:{width:"75px"},attrs:{type:"number"},on:{change:function(t){return e.editConfig(t,"future_test_notice_limit_min")}},model:{value:e.config.tradeFutureTestNoticeLimitMin,callback:function(t){e.$set(e.config,"tradeFutureTestNoticeLimitMin",t)},expression:"config.tradeFutureTestNoticeLimitMin"}})],1)])])],2),a("el-collapse-item",{attrs:{name:"spot"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.spotsTrade"))+"("+e._s(e.$t("trade.notYet"))+"): ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"spot_enable")}},model:{value:e.config.tradeSpotEnable,callback:function(t){e.$set(e.config,"tradeSpotEnable",t)},expression:"config.tradeSpotEnable"}})],1)]),a("div",{staticClass:"dashboard-text"},[a("div",{staticStyle:{"margin-left":"20px"}},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.websocket"))+": ")]),a("el-switch",{attrs:{value:e.config.wsSpotEnable,"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"ws_spot_enable")}}}),a("span",{staticClass:"green",staticStyle:{"margin-left":"20px"}},[e._v(e._s(e.$t("showPage.autoUpdatePrice")))])],1)])])],2),a("el-collapse-item",{attrs:{name:"delivery"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.deliveryTrade"))+"("+e._s(e.$t("trade.notYet"))+"): ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"delivery_enable")}},model:{value:e.config.tradeDeliveryEnable,callback:function(t){e.$set(e.config,"tradeDeliveryEnable",t)},expression:"config.tradeDeliveryEnable"}})],1)]),a("div",{staticClass:"dashboard-text"},[a("div",{staticStyle:{"margin-left":"20px"}},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.websocket"))+": ")]),a("el-switch",{attrs:{value:e.config.wsDeliveryEnable,"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"ws_delivery_enable")}}}),a("span",{staticClass:"green",staticStyle:{"margin-left":"20px"}},[e._v(e._s(e.$t("showPage.autoUpdatePrice")))])],1)])])],2),a("el-collapse-item",{attrs:{name:"new_coin_rush"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.newCoinRush"))+": ")])])]),a("div",{staticClass:"dashboard-text"},[a("div",{staticStyle:{"margin-left":"20px"}},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.newSpotRush"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"spot_new_enable")}},model:{value:e.config.spotNewEnable,callback:function(t){e.$set(e.config,"spotNewEnable",t)},expression:"config.spotNewEnable"}})],1),a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.newFuturesRush"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"future_new_enable")}},model:{value:e.config.tradeNewEnable,callback:function(t){e.$set(e.config,"tradeNewEnable",t)},expression:"config.tradeNewEnable"}})],1)])])],2),a("el-collapse-item",{attrs:{name:"coin_notice"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.coinNotice"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"notice_coin_enable")}},model:{value:e.config.noticeCoinEnable,callback:function(t){e.$set(e.config,"noticeCoinEnable",t)},expression:"config.noticeCoinEnable"}})],1)])],2),a("el-collapse-item",{attrs:{name:"market_listen"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.marketListen"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"listen_coin_enable")}},model:{value:e.config.listenCoinEnable,callback:function(t){e.$set(e.config,"listenCoinEnable",t)},expression:"config.listenCoinEnable"}})],1)])],2),a("el-collapse-item",{attrs:{name:"funding_rate"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("route.fundingRate"))+": ")]),a("el-switch",{attrs:{"active-color":"#13ce66","inactive-color":"#dcdfe6","active-value":1,"inactive-value":0},on:{change:function(t){return e.editConfig(t,"listen_funding_rate_enable")}},model:{value:e.config.listenFundingRate,callback:function(t){e.$set(e.config,"listenFundingRate",t)},expression:"config.listenFundingRate"}})],1)])],2),a("el-collapse-item",{attrs:{name:"debug"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v("debug: ")]),"1"===e.config.debug?a("span",{staticClass:"red"},[e._v(e._s(e.$t("showPage.open")))]):a("span",{staticClass:"green"},[e._v(e._s(e.$t("showPage.close")))])])]),a("div",{staticClass:"dashboard-text"},[a("div",{staticStyle:{"margin-left":"20px"}},[a("div",{staticClass:"dashboard-text"},[a("el-button",{attrs:{size:"mini",type:"primary"},on:{click:e.testPusher}},[e._v(e._s(e.$t("showPage.testPusher")))])],1)])])],2),a("el-collapse-item",{attrs:{name:"external"}},[a("template",{slot:"title"},[a("div",{staticClass:"dashboard-text"},[a("span",[e._v(e._s(e.$t("showPage.externalLinks"))+": ")])])]),a("div",{staticClass:"dashboard-text"},[a("div",{staticStyle:{"margin-left":"20px",display:"flex",gap:"20px"}},e._l(e.config.externalLinks,(function(t){return a("el-link",{key:t.title,staticStyle:{"font-size":"30px"},attrs:{type:"primary",underline:!1,href:t.url,target:"_blank"}},[e._v(" "+e._s(t.title)+" ")])})),1)])],2)],1)],1)},i=[],s=a("dcb1"),o=a("dd36"),c=a("fee1"),r=(a("e224"),a("4cc3"),a("374d"),a("5227"),a("bd1a"),a("b775"));function l(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(r["a"])({url:"/service/config",method:"get",params:e})}function u(e){return Object(r["a"])({url:"/service/config",method:"put",data:e})}function d(){var e=arguments.length>0&&void 0!==arguments[0]?arguments[0]:{};return Object(r["a"])({url:"/test-pusher",method:"post",params:e})}var f=a("5f87"),v=a("2465"),g={name:"Dashboard",data:function(){return{loading:!1,symbols:[],excludeSymbols:[],config:{debug:"0",coinAllowLong:1,coinAllowShort:0,coinExcludeSymbols:"",coinMaxCount:0,coinOrderType:"",externalLinks:[],listenCoinEnable:0,listenFundingRate:0,noticeCoinEnable:0,spotNewEnable:0,tradeFutureTest:0,tradeFutureTestNoticeLimitMin:0,tradeFutureEnable:0,tradeSpotEnable:0,tradeDeliveryEnable:0,tradeNewEnable:0,tradeStrategyCoin:"",tradeStrategyTrade:"",wsFuturesEnable:0,wsSpotEnable:0,wsDeliveryEnable:0,futuresPositionConvertEnable:0}}},created:function(){var e=this;return Object(c["a"])(Object(o["a"])().mark((function t(){return Object(o["a"])().wrap((function(t){while(1)switch(t.prev=t.next){case 0:e.getSymbols(),e.fetchConfig();case 2:case"end":return t.stop()}}),t)})))()},methods:{fetchConfig:function(){var e=this;return Object(c["a"])(Object(o["a"])().mark((function t(){var a,n;return Object(o["a"])().wrap((function(t){while(1)switch(t.prev=t.next){case 0:return t.next=2,l();case 2:a=t.sent,n=a.data;try{n.externalLinks=JSON.parse(n.externalLinks),e.excludeSymbols=""===n.coinExcludeSymbols.trim()?[]:n.coinExcludeSymbols.split(",")}catch(i){n.externalLinks=[]}Object(f["d"])(n),e.config=n;case 7:case"end":return t.stop()}}),t)})))()},editConfig:function(e,t){var a=this;return Object(c["a"])(Object(o["a"])().mark((function n(){return Object(o["a"])().wrap((function(n){while(1)switch(n.prev=n.next){case 0:return a.loading=!0,n.prev=1,"future_max_count"!==t&&"future_test_notice_limit_min"!==t||(e=Number(e)),"future_exclude_symbols"===t&&(e=a.excludeSymbols.join(",")),n.next=6,u(Object(s["a"])({},t,e));case 6:return n.next=8,a.fetchConfig();case 8:a.$message({message:a.$t("table.actionSuccess"),type:"success"}),n.next=14;break;case 11:n.prev=11,n.t0=n["catch"](1),a.$message({message:a.$t("table.actionFail"),type:"error"});case 14:return n.prev=14,a.loading=!1,n.finish(14);case 17:case"end":return n.stop()}}),n,null,[[1,11,14,17]])})))()},testPusher:function(){var e=this;return Object(c["a"])(Object(o["a"])().mark((function t(){return Object(o["a"])().wrap((function(t){while(1)switch(t.prev=t.next){case 0:return t.prev=0,t.next=3,d();case 3:e.$message({message:e.$t("table.actionSuccess"),type:"success"}),t.next=9;break;case 6:t.prev=6,t.t0=t["catch"](0),e.$message({message:e.$t("table.actionFail"),type:"error"});case 9:case"end":return t.stop()}}),t,null,[[0,6]])})))()},getSymbols:function(){var e=this;return Object(c["a"])(Object(o["a"])().mark((function t(){var a,n;return Object(o["a"])().wrap((function(t){while(1)switch(t.prev=t.next){case 0:return t.next=2,Object(v["f"])();case 2:a=t.sent,n=a.data,e.symbols=n.map((function(e){return e.symbol}));case 5:case"end":return t.stop()}}),t)})))()}}},p=g,b=(a("5fe0"),a("9bf6")),h=Object(b["a"])(p,n,i,!1,null,"9035b5d2",null);t["default"]=h.exports}}]);