import { ref } from 'vue'

const isDark = ref(false)

export const useTheme = () => {
  const toggleTheme = () => {
    isDark.value = !isDark.value
    document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
  }
  return { isDark, toggleTheme }
}
