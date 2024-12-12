CREATE TABLE `futures_orders` (
    `id` integer NOT NULL primary key autoincrement,
    `symbol` varchar(255),
	`client_order_id` varchar(255),
	`order_id` varchar(255),
    `side` varchar(255),
    `position_side` varchar(255),
    `type` varchar(255),
    `status` varchar(255),
    `price` varchar(255),
    `orig_qty` varchar(255),
    `executed_qty` varchar(255) DEFAULT ('0'),
   
    `createTime` integer,
    `updateTime` integer
);