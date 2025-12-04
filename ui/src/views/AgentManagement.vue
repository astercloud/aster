<template>
  <div class="agent-management">
    <div v-if="!selectedAgent" class="dashboard-view">
      <AgentDashboard @chat="handleSelectAgent" />
    </div>
    <div v-else class="chat-view">
      <AgentChatSession :agent="selectedAgent" @back="selectedAgent = null" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { AgentDashboard } from "@/components/Agent";
import AgentChatSession from "@/components/Agent/AgentChatSession.vue";
import type { Agent } from "@/types";

const selectedAgent = ref<Agent | null>(null);

const handleSelectAgent = (agent: Agent) => {
  selectedAgent.value = agent;
};
</script>

<style scoped>
.agent-management {
  @apply min-h-screen bg-background dark:bg-background-dark;
}

.dashboard-view {
  @apply p-8;
}

.chat-view {
  @apply h-screen;
}
</style>
