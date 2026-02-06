## Description
<!-- Describe your changes in detail -->

## Type of change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## How Has This Been Tested?
<!-- Describe the tests you ran -->

## Checklist:
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes

## Related Issue
<!-- Link to the issue: Fixes #123 -->
```

---

### **Phase 6: Setup GitHub Secrets**

You need to add secrets for Docker Hub:

**Via Web:**
1. Go to: `https://github.com/YOUR_USERNAME/instvisor/settings/secrets/actions`
2. Click "New repository secret"
3. Add these secrets:
```
Name: DOCKERHUB_USERNAME
Value: your-dockerhub-username

Name: DOCKERHUB_TOKEN
Value: your-dockerhub-access-token