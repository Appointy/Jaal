## A brief history of Jaal

This document lists the challenges which led to adoption of GraphQL technology and development of Jaal.

### Challenges

* *Large amount of data*: One of the main problems with the traditional REST APIs are that it either returns too much data or too little data. Thus, it becomes difficult for UI to handle this large amount of data.
* *Repetitive code*: To collect a variety of data, UI makes multiple calls to server. This code is repetitive in nature and error prone.
* *Network Latency*: Multiple calls to server resulted in a lot of network latency. Hence, the system appears slow.

### Solution
[GraphQL](https://graphql.org/) is a query language for APIs. It exposes a single endpoint for data fetching and allows UI to define their data requirements. Hence, a GraphQL server overcomes all the above mentioned challenges as it only returns the data required. 

### Goal
The goal was to develop spec compliant GraphQL server without disturbing Appointy's tech stack. There are various frameworks to develop GraphQL servers but none of them are spec compliant and had many different drawbacks. 

To achieve our goal, we developed Jaal - A spec compliant GraphQL server framework. Jaal exposes various APIs to develop spec compliant GraphQL servers. Along with Jaal, we developed *protoc-gen-jaal* - a protocol buffer plugin to generate the code required to develop a GraphQL server.
