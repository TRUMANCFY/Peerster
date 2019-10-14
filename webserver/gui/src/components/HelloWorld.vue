<template>
  <div class="hello">
    <b-container class="bv-example-row">
  <b-row>
    <b-col>
      <b-form>
        <div class="form-group">
          <label>Chat Box</label>
          <textarea class="form-control" id="ChatBox" rows="3" v-model="peerMsgs" readonly></textarea>
        </div>

        <div class="form-group">
          <label>Node Box</label>
          <textarea class="form-control" id="NodeBox" rows="5" v-model="knownNodes" readonly></textarea>
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
        >{{msgToSend}}</b-form-input>
        <br>
        <b-button class='float-right' @click="submitMsg" >Submit</b-button>
        <br>
        <br>
        <br>
        <label>Add New Peer</label>
        <b-form-input
          id="NewPeer"
          placeholder="address:port"
        >{{newPeer}}</b-form-input>
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
// Button IDS: 

export default {
  name: 'HelloWorld',
  props: {

  },
  data: function() {
    return {
      knownNodes: [1,2,3],
      peerID: 100,
      peerMsgs: [],
      newPeer: "",
      msgToSend: ""
    }
  },
  methods: {
    submitMsg: function() {

    },
    
    addPeer: function() {
      
    }
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>

</style>


