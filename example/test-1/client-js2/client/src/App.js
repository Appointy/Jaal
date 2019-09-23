import React from "react"
import sub from './Sub';
import './App.css';
import gql from 'graphql-tag'

function App() {
  sub.subscribe({
    query: gql`
      subscription test{
          channelStream(in: {name: "Serial Killer"}) {
            name
            email
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
