import React from "react"
import sub from './Sub';
import './App.css';
import gql from 'graphql-tag'

function App() {
  sub.subscribe({
    query: gql`
      subscription test1{
          postStream(in: {tag: "Huer"}) {
            title
          }
          channelStream(in: {name: "Serial Killer"}) {
            email
            name
          }
      }`,
    variables: {}
  }).subscribe({
    next(data) {
      console.log(data)
    }
  });
  return (
    <div className="App">
    </div>
  );
}

export default App;
