<template>
  <div class="hello">
    <b-container class="bv-example-row">
  <b-row>
    <b-col>
      <b-form>
        <div class="form-group">
          <label>Chat Box</label>
          <textarea class="form-control" id="ChatBox" rows="10" v-model="peerMsgStr" readonly></textarea>
        </div>

        <div class="form-group">
          <label>Node Box</label>
          <textarea class="form-control" id="NodeBox" rows="5" v-model="knownNodesStr" readonly></textarea>
        </div>

        <div class="form-group" >
          <label>Peer ID</label>
          <textarea class="form-control" id="PeerID" rows="1" v-model="peerID" readonly></textarea>
        </div>
      </b-form>
    </b-col>
    <b-col>
    <b-form>
      <label>Please provide the Message</label>
      <b-form-input
          id="Message"
          placeholder="Message"
          v-model="msgToSend"
        ></b-form-input>
        <br>
        <b-button class='float-right' @click="submitMsg" >Submit</b-button>
        <br>
        <br>
        <br>
        <label>Add New Peer</label>
        <b-form-input
          id="NewPeer"
          placeholder="address:port"
          v-model="newPeer"
        ></b-form-input>
        <br>
        <b-button class='float-right' @click="addPeer">Add Peer</b-button>
    </b-form>
    </b-col>
  </b-row>
</b-container>
  </div>
</template>

<script>
import Vue from 'vue'
import BootstrapVue from 'bootstrap-vue'

import 'bootstrap/dist/css/bootstrap.css'
import 'bootstrap-vue/dist/bootstrap-vue.css'

Vue.use(BootstrapVue)


// Container IDs: ChatBox/NodeBox/PeerID
// Button IDS

export default {
  name: 'HelloWorld',
  props: {

  },
  data: function() {
    return {
      knownNodes: [],
      knownNodesStr: "",
      peerID: "",
      peerMsg: [],
      peerMsgStr: "",      
      newPeer: "",
      msgToSend: ""
    }
  },
  methods: {
    submitMsg: async function() {
      var self = this
      console.log(self.msgToSend)

      if (self.msgToSend == "") {
        alert("Please input the message")
        return
      }

      var payload = {"Text": self.msgToSend}

      var a = await fetch("/message", {method: "POST", body: JSON.stringify(payload), mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var temp = res.json()
          return temp
        }
      })

      console.log(a)

      if (a) {
        self.msgToSend = ""
      }
    },
    
    addPeer: async function() {
      var self = this

      console.log(self.newPeer)

     if (self.newPeer == "") {
       alert("Please input the new peer")
       return
     }

      // test the communicatioon
      var payload = {"Addr": self.newPeer}
      
      var a = await fetch('/node', {method: "POST", body: JSON.stringify(payload), mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var tmp = res.json()
          return tmp
        }
      })

      if (a) {
        self.newPeer = ""
      }

    },
    sortMsgStr: function(s1, s2) {
      var as = s1.toLowerCase();
      var bs = s2.toLowerCase();
      if (as < bs) return -1;
      if (as > bs) return 1;
    },
    updateOtherComp: function() {
      var self = this
      var tmpNodes = []
      
      self.knownNodes["nodes"].map(a => {
        tmpNodes.push(a)
      })

      tmpNodes = tmpNodes.sort()

      self.knownNodesStr = tmpNodes.join('\n')

      var tmpMsgSorted = self.peerMsg['messages'].sort((a, b) => {
        if (a.Origin == b.Origin) {
          return a.ID - b.ID
        }
        else {
          return self.sortMsgStr(a.Origin, b.Origin)
        }
      })
      var tmpMsg = []
      tmpMsgSorted.map(a => {
        tmpMsg.push(a.Origin + " " + a.ID + " " + a.Text + '\n')
      })
      self.peerMsgStr = tmpMsg.join('\n')
    },

    pullMessage: async function() {
      var self = this
      self.peerMsg = await fetch('/message', {method: 'GET', mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var tmp = res.json();
          return tmp;
        }
      })
      .catch(err => {
        console.log(err)
      })

      console.log(self.peerMsg)
    },

    pullNodes: async function() {
      var self = this
      self.knownNodes = await fetch('/node', {method: 'GET', mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var tmp = res.json()
          return tmp
        }
      })
      .catch(err=>{
          console.log(err)
      })

      console.log(self.knownNodes)
    },
    pullPeerID: async function() {
      var self = this
      self.peerID = await fetch('/id', {method: 'GET', mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var tmp = res.json()
          return tmp
        }
      })
      .catch(err => {
        console.log(err)
      })
      console.log(self.peerID)
      self.peerID = self.peerID.id

      console.log(self.peerID)
      
    },

    refresh: function() {
      console.log('refresh')
      this.pullMessage()
      this.pullPeerID()
      this.pullNodes()
      this.updateOtherComp()
    }
  },
  mounted() {
    // setInterval(this.refresh, 1000)
    setInterval(this.refresh, 1000)
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>

</style>



