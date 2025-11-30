<template>
  <div :class="['loading-spinner', sizeClass]">
    <svg
      :class="['spinner-icon', colorClass]"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="spinner-track"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      ></circle>
      <path
        class="spinner-path"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
    <span v-if="text" :class="['spinner-text', textSizeClass]">{{ text }}</span>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';

type SpinnerSize = 'sm' | 'md' | 'lg';
type SpinnerColor = 'primary' | 'secondary' | 'white';

export default defineComponent({
  name: 'LoadingSpinner',

  props: {
    size: {
      type: String as () => SpinnerSize,
      default: 'md',
    },
    color: {
      type: String as () => SpinnerColor,
      default: 'primary',
    },
    text: {
      type: String,
      default: undefined,
    },
  },

  setup(props) {
    const sizeClass = computed(() => {
      const sizes: Record<SpinnerSize, string> = {
        sm: 'spinner-sm',
        md: 'spinner-md',
        lg: 'spinner-lg',
      };
      return sizes[props.size];
    });

    const colorClass = computed(() => {
      const colors: Record<SpinnerColor, string> = {
        primary: 'text-primary',
        secondary: 'text-secondary dark:text-secondary-dark',
        white: 'text-white',
      };
      return colors[props.color];
    });

    const textSizeClass = computed(() => {
      const sizes: Record<SpinnerSize, string> = {
        sm: 'text-xs',
        md: 'text-sm',
        lg: 'text-base',
      };
      return sizes[props.size];
    });

    return {
      sizeClass,
      colorClass,
      textSizeClass,
    };
  },
});
</script>

<style scoped>
.loading-spinner {
  @apply flex flex-col items-center justify-center gap-2;
}

.spinner-icon {
  @apply animate-spin;
}

.spinner-sm .spinner-icon {
  @apply w-4 h-4;
}

.spinner-md .spinner-icon {
  @apply w-8 h-8;
}

.spinner-lg .spinner-icon {
  @apply w-12 h-12;
}

.spinner-track {
  @apply opacity-25;
}

.spinner-path {
  @apply opacity-75;
}

.spinner-text {
  @apply text-secondary dark:text-secondary-dark;
}
</style>
