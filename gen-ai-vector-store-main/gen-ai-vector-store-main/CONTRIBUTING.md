The Lean fnx service project follows the [innersource](https://resources.github.com/whitepapers/introduction-to-innersource/) development model where contributors across Pega development maintain aspects of the project.  This document provides the guidelines for contributing code to this repository.

# Guiding Principles

This project showcases a production-grade typical Java based REST service.  The service will be a slimmed down opinionated example of such a service.  It will *not* be a showcase of all technology that is supported but just tools to provide coverage of the lean fnx service that is required for it to be fully validated as a production quality service.

# Roles

The following are the roles defined in this project.

## Maintainers

Code owners who are responsible for driving the vision and managing the organizational aspects of the project. They are not necessarily the original owners or authors of the code.

These people own portions of the code base and must review contributions made to their respective areas.  At least one person from the group must approve pull requests.

The ownership of the code is represented in the [CODEOWNERS](CODEOWNERS) file in this repository.

### Ongoing Support

Maintaining groups *must* participate in operational support for community members
  - Need to have a published team rotation available across regions during business hours
  - Need to be subscribed to the *Platform Services Support* webex space and respond to queries
  - Every support question needs an answer

## Contributors

Everyone who has contributed something back to the project. Code contributed is still owned by maintainers of that area.

As a contributor interested in improving the project, the first step is to reach out to the maintainers group to present the contribution change if of significance.  Then the contributor will create a branch and push that branch to the main repository.  The contributor will open a pull request that would automatically add the correct maintainers to review the change and who can then decide to accept or decline it.  

## Community Members

People who use the project. They might be active in conversations or express their opinion on the project’s direction.

As a user of the template, a community member is expected to contribute enhancements and bug fixes as they see gaps or issues in the template so that all can benefit.
 
# Backlog / Issue Management

We have a product, PRD-6797, for the Lean FNX Service in Agile Studio to track in-progress and future planned work.  Bugs / Issues will be maintained in BL-9958.  

# Communication

Maintainers shall communicate within the *Lean Fnx Service Developers* webex teams space

The maintainers will meet on a monthly basis to discuss the status of the project and the next set of future contributions and any change to the vision of the project.  All contributors and community members are welcome to attend this meeting as well.

# Release Management

The Lean Fnx Service project will use the SDEA Release Management process codified in the Jenkins Library Pipeline template.

The [RELEASE_NOTES.md](RELEASE_NOTES.md) must be kept up to date with the latest changes of the template so that community members can update their own repositories with new features of the lean fnx service template.

When a release is available, it shall announced in the Webex Teams space XXXXX where community members are suscribed and can adopt the new capabilities.

# Outstanding Questions

* How do we ensure that contributions outside of the maintaining groups actually occur?  Carrot / Stick approach?  Monthly recognition? 
