FROM scratch

ENV USER_UID=1001 PATH=/

# install operator binary
COPY bin/submariner-operator /submariner-operator

ENTRYPOINT ["/submariner-operator"]

USER ${USER_UID}
