<template>
  <div class="hello">
    <b-container class="bv-example-row">
  <b-row>
    <b-col>
      <b-form>
        <div class="form-group">
          <label>Chat Box</label>
          <textarea class="form-control" id="ChatBox" rows="5" v-model="peerMsgStr" readonly></textarea>
        </div>

         <div class="form-group">
          <label>Private Message</label>
          <textarea class="form-control" id="PrivateBox" rows="5" v-model="privateMsgStr" readonly></textarea>
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
      <br>
      <br>
      <br>
      <div class="form-group">
        <div>
          <label>Private Message</label>
        </div>
        <div style="width:20%;float:left">
          <select v-model="peerSelected" style="width:100%" multiple>
              <option v-for="(peer,index) in originTarget" :key="`peer-${index}`">
                {{ peer }}
              </option>
          </select>
        </div>
        <div style="width:80%;float:right;">
          <div style="height:50%">
          <b-form-input
          id="PrivateMessage"
          placeholder="Private Message"
          v-model="privateMsgToSend"
          style="width:80%;margin: 0 auto"
        ></b-form-input>
          </div>
          <div style="height:50%">
            <b-button class='float-center' style="margin-top: 5%" @click="submitPrivateMsg">Send Private</b-button>
          </div>
        </div>
      </div>
    
      <div style="margin-top: 8em">
        <label>Select file to share</label>
        <b-form-file
        v-model="file"
        :state="Boolean(file)"
        placeholder="Choose a file or drop it here..."
        drop-placeholder="Drop file here..."
      ></b-form-file>
      <b-button class='float-center' style="margin-top:1em" @click="sendFile">Send File</b-button>
      </div>

       
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
      msgToSend: "",
      peerSelected: [],
      originTarget: [],
      privateMsgToSend: "",
      privateMsg: [],
      privateMsgStr: "",
      file: null,
      regExp: /^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]):[0-9]+$/
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

      if (a) {
        self.msgToSend = ""
      }
    },

    submitPrivateMsg: async function() {
      var self = this;
      console.log('Send the private message')
      if (self.peerSelected.length == 0) {
        alert('Please select the valid target')
        return
      }

      if (self.privateMsgToSend == '') {
        alert("Please input the valid private message!")
        return
      }

      var dest = self.peerSelected[0]

      var payload = {
        "Text": self.privateMsgToSend,
        "Dest": dest
      }

      var a = await fetch("/private", {method: "POST", body: JSON.stringify(payload), mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var temp = res.json()
          return temp
        }
      })

      console.log(a)

      if (a) {
        self.privateMsgToSend = ""
        self.peerSelected = []
      }

    },
    
    addPeer: async function() {
      var self = this

      console.log(self.newPeer)

     if (self.newPeer == "") {
       alert("Please input the new peer")
       return
     }

     if (!self.matchExact(self.regExp, self.newPeer)) {
       alert("Please input the valid address")
       self.newPeer = ""
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
    matchExact: function(r, str) {
      var match = str.match(r);
      return match && str === match[0];
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
      });

      var tmpMsg = [];
      tmpMsgSorted.map(a => {
        tmpMsg.push(a.Origin + " " + a.ID + " " + a.Text + '\n')
      });
      
      self.peerMsgStr = tmpMsg.join('\n');

      // Deal with PrivateMessage
      var tmpPrivateMsgSorted = self.privateMsg.sort((a, b) => {
        if (a.Origin == b.Origin) {
          return self.sortMsgStr(a.Text, b.Text)
        }
        else {
          return self.sortMsgStr(a.Origin, b.Origin)
        }
      })

      console.log(tmpPrivateMsgSorted)
      
      var tmpPrivateMsg = [];

      tmpPrivateMsgSorted.map(a => {
        tmpPrivateMsg.push(a.Origin + " " + a.Text)
      })

      console.log(tmpMsgSorted)

      self.privateMsgStr = tmpPrivateMsg.join("\n")

      console.log(self.privateMsgStr)
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

    pullPrivateMsg: async function() {
      // Get the private message
      var self = this;

      self.privateMsg = await fetch('/private', {method: 'GET', mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var tmp = res.json();
          return tmp;
        }
      })
      .catch(err => {
        console.log(err)
      })

      self.privateMsg = self.privateMsg.msgs;

      console.log(self.privateMsg)
    },

    pullRouteTarget: async function() {
      // Get available target
      var self = this;

      self.originTarget = await fetch('/routes', {method: 'GET', mode: 'cors'})
      .then(res => {
        if (res.ok) {
          var tmp = res.json();
          return tmp;
        }
      })
      .catch(err => {
        console.log(err)
      })

      self.originTarget = self.originTarget.targets;

      // Sort the targets
      self.originTarget = self.originTarget.sort((a, b) => {
        return self.sortMsgStr(a, b)
      })

      console.log(self.originTarget)

    },

    sendFile: function() {
      var self = this;
      console.log(self.file)
    },

    refresh: function() {
      console.log('refresh')
      // this.pullMessage()
      // this.pullPeerID()
      // this.pullNodes()
      // this.pullPrivateMsg()
      // this.pullRouteTarget()
      // this.updateOtherComp()
    }
  },
  mounted() {
    // setInterval(this.refresh, 1000)
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>

</style>



