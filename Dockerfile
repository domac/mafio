FROM hub.c.163.com/library/alpine:latest

MAINTAINER domac.li

EXPOSE 10630

ADD releases/mafio /mafio
ADD releases/agent.cfg /agent.conf
ADD releases/agent.sh /agent.sh

RUN chmod +x /mafio
RUN chmod a+x /agent.sh

EXPOSE 10630
ENTRYPOINT ["/agent.sh"]