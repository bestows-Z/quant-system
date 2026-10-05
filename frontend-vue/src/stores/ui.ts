import { defineStore } from 'pinia'

export const useUiStore = defineStore('ui', {
  state: () => ({ mobileMenuOpen: false }),
  actions: {
    toggleMobileMenu() { this.mobileMenuOpen = !this.mobileMenuOpen },
    closeMobileMenu() { this.mobileMenuOpen = false }
  }
})
