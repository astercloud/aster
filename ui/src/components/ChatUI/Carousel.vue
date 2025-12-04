<template>
  <div class="carousel">
    <div class="carousel-container" ref="containerRef">
      <div class="carousel-track" :style="{ transform: `translateX(-${currentIndex * 100}%)` }">
        <div v-for="(item, index) in items" :key="index" class="carousel-item">
          <slot :item="item" :index="index">
            {{ item }}
          </slot>
        </div>
      </div>
    </div>

    <button v-if="showArrows && currentIndex > 0" class="carousel-arrow carousel-prev" @click="prev">
      <Icon type="chevron-left" />
    </button>
    <button v-if="showArrows && currentIndex < items.length - 1" class="carousel-arrow carousel-next" @click="next">
      <Icon type="chevron-right" />
    </button>

    <div v-if="showDots" class="carousel-dots">
      <button v-for="(_, index) in items" :key="index" :class="['carousel-dot', { active: index === currentIndex }]" @click="goTo(index)" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import Icon from "./Icon.vue";

interface Props {
  items: any[];
  showArrows?: boolean;
  showDots?: boolean;
  autoplay?: boolean;
  interval?: number;
}

const props = withDefaults(defineProps<Props>(), {
  showArrows: true,
  showDots: true,
  autoplay: false,
  interval: 3000,
});

const currentIndex = ref(0);
const containerRef = ref<HTMLDivElement>();

const prev = () => {
  if (currentIndex.value > 0) {
    currentIndex.value--;
  }
};

const next = () => {
  if (currentIndex.value < props.items.length - 1) {
    currentIndex.value++;
  }
};

const goTo = (index: number) => {
  currentIndex.value = index;
};
</script>

<style scoped>
.carousel {
  @apply relative overflow-hidden;
}

.carousel-container {
  @apply relative overflow-hidden;
}

.carousel-track {
  @apply flex transition-transform duration-300 ease-in-out;
}

.carousel-item {
  @apply flex-shrink-0 w-full;
}

.carousel-arrow {
  @apply absolute top-1/2 -translate-y-1/2 z-10 p-2 bg-white/80 dark:bg-gray-800/80 hover:bg-white dark:hover:bg-gray-800 rounded-full shadow-lg transition-all;
}

.carousel-prev {
  @apply left-4;
}

.carousel-next {
  @apply right-4;
}

.carousel-dots {
  @apply absolute bottom-4 left-1/2 -translate-x-1/2 flex gap-2 z-10;
}

.carousel-dot {
  @apply w-2 h-2 rounded-full bg-white/50 hover:bg-white/80 transition-colors;
}

.carousel-dot.active {
  @apply bg-white;
}
</style>
